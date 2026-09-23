package configcue

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/pbnjay/memory"

	"cuelang.org/go/cue/ast"
	"github.com/lucasew/workspaced/internal/git"
	"github.com/lucasew/workspaced/internal/modulecue"
	"github.com/lucasew/workspaced/pkg/driver"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
	"github.com/lucasew/workspaced/pkg/filespine"
	"github.com/lucasew/workspaced/pkg/logging"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"
)

var (
	ErrInvalidInputSpec = errors.New("invalid input spec")
	ErrDecodeInputs     = errors.New("decode inputs for resolution")
)

//go:embed schema.cue prelude_common.cue prelude_home.cue prelude_codebase.cue
var schemaFS embed.FS

type Layer struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type DiscoverOptions struct {
	Cwd string
	// Mode is injected as runtime.mode (home, codebase, system).
	Mode string
	// HomeLayers uses home discovery and prelude_home. Independent of Mode
	// so codebase apply in $DOTFILES can keep home layers.
	HomeLayers bool
}

func (o DiscoverOptions) RuntimeMode() string {
	if o.Mode != "" {
		return o.Mode
	}
	return filespine.ModeCodebase
}

type DiscoverResult struct {
	Layers []Layer `json:"layers"`
}

type EvaluationResult struct {
	Layers []Layer         `json:"layers"`
	JSON   json.RawMessage `json:"json"`
	Value  cue.Value       `json:"-"`
}

func DiscoverLayers(ctx context.Context, opts DiscoverOptions) (DiscoverResult, error) {
	layers := make([]Layer, 0)

	if !opts.HomeLayers {
		repoPath, err := ResolveWorkspaceCuePath(ctx, opts.Cwd)
		if err != nil {
			return DiscoverResult{}, err
		}
		if repoPath != "" {
			layers = append(layers, Layer{Name: "repo", Path: repoPath})
		}
	}

	if opts.HomeLayers {
		dotfilesRoot, err := envdriver.GetDotfilesRoot(ctx)
		if err == nil && dotfilesRoot != "" {
			p := filepath.Join(dotfilesRoot, "workspaced.cue")
			if fileExists(p) {
				layers = append(layers, Layer{Name: "dotfiles", Path: p})
			}
		}
	}

	if opts.HomeLayers {
		homeDir, err := envdriver.ResolveHomeDir()
		if err == nil && homeDir != "" {
			p := filepath.Join(homeDir, "workspaced.cue")
			if fileExists(p) {
				layers = append(layers, Layer{Name: "user", Path: p})
			}
		}
	}

	if opts.HomeLayers {
		configDir, err := envdriver.GetConfigDir(ctx)
		if err == nil && configDir != "" {
			p := filepath.Join(configDir, "workspaced.cue")
			if fileExists(p) {
				layers = append(layers, Layer{Name: "home", Path: p})
			}
		}
	}

	return DiscoverResult{Layers: layers}, nil
}

func ExportJSON(ctx context.Context, opts DiscoverOptions) ([]byte, error) {
	result, err := Evaluate(ctx, opts)
	if err != nil {
		return nil, err
	}
	return result.JSON, nil
}

func ExportCUE(ctx context.Context, opts DiscoverOptions) ([]byte, error) {
	// Use a fresh root with logger for the (rare) diagnostic warnings in this
	// top-level export path. The real work ctx is not threaded into these
	// high-level CUE export helpers.
	return exportFormatted(ctx, opts, formatWorkspacedValue)
}

func ExportDef(ctx context.Context, opts DiscoverOptions) ([]byte, error) {
	return exportFormatted(ctx, opts, formatWorkspacedDef)
}

func exportFormatted(ctx context.Context, opts DiscoverOptions, format func(context.Context, cue.Value, []string, []Layer) ([]byte, error)) ([]byte, error) {
	paths, layers, err := discoverPaths(ctx, opts)
	if err != nil {
		return nil, err
	}
	configValue, err := buildWorkspacedValue(ctx, paths, layers, opts)
	if err != nil {
		return nil, err
	}
	return format(ctx, configValue, paths, layers)
}

func Evaluate(ctx context.Context, opts DiscoverOptions) (EvaluationResult, error) {
	paths, layers, err := discoverPaths(ctx, opts)
	if err != nil {
		return EvaluationResult{}, err
	}
	configValue, err := buildWorkspacedValue(ctx, paths, layers, opts)
	if err != nil {
		return EvaluationResult{}, err
	}
	b, err := marshalWorkspacedValue(ctx, configValue, paths, layers)
	if err != nil {
		return EvaluationResult{}, err
	}
	return EvaluationResult{
		Layers: layers,
		JSON:   b,
		Value:  configValue,
	}, nil
}

// discoverPaths loads config layers and returns their paths in the same order.
func discoverPaths(ctx context.Context, opts DiscoverOptions) (paths []string, layers []Layer, err error) {
	discovered, err := DiscoverLayers(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	paths = make([]string, 0, len(discovered.Layers))
	for _, layer := range discovered.Layers {
		paths = append(paths, layer.Path)
	}
	return paths, discovered.Layers, nil
}

func ExportJSONFromPaths(ctx context.Context, paths []string) ([]byte, error) {
	return exportJSONFromPaths(ctx, paths, nil, false)
}

func exportJSONFromPaths(ctx context.Context, paths []string, discovered []Layer, homeLayers bool) ([]byte, error) {
	opts := DiscoverOptions{HomeLayers: homeLayers, Mode: filespine.ModeCodebase}
	if homeLayers {
		opts.Mode = filespine.ModeHome
	}
	configValue, err := buildWorkspacedValue(ctx, paths, discovered, opts)
	if err != nil {
		return nil, err
	}
	return marshalWorkspacedValue(ctx, configValue, paths, discovered)
}

func buildWorkspacedValue(ctx context.Context, paths []string, discovered []Layer, opts DiscoverOptions) (cue.Value, error) {
	mode := opts.RuntimeMode()
	homeLayers := opts.HomeLayers
	baseRuntimePrelude, err := buildRuntimePrelude(ctx, nil, mode)
	if err != nil {
		return cue.Value{}, err
	}
	cueCtx := cuecontext.New()
	initialValue, err := compileWorkspacedValueWithContext(cueCtx, paths, baseRuntimePrelude, homeLayers, nil, nil)
	if err != nil {
		return cue.Value{}, err
	}
	resolvedInputs, err := resolveRuntimeInputs(initialValue, paths, discovered)
	if err != nil {
		return cue.Value{}, err
	}
	runtimePrelude, err := buildRuntimePrelude(ctx, resolvedInputs, mode)
	if err != nil {
		return cue.Value{}, err
	}
	baseConfigValue, err := compileWorkspacedValueWithContext(cueCtx, paths, runtimePrelude, homeLayers, nil, nil)
	if err != nil {
		return cue.Value{}, err
	}
	preLayers, postLayers, err := buildResolvedModuleLayers(baseConfigValue, paths, discovered)
	if err != nil {
		return cue.Value{}, err
	}
	configValue, err := compileWorkspacedValueWithContext(cueCtx, paths, runtimePrelude, homeLayers, preLayers, postLayers)
	if err != nil {
		return cue.Value{}, err
	}
	fileLayers, err := buildModuleFileLayers(configValue, paths, discovered)
	if err != nil {
		return cue.Value{}, err
	}
	if len(fileLayers) == 0 {
		return configValue, nil
	}
	postWithFiles := append(append([]compiledLayer{}, postLayers...), fileLayers...)
	return compileWorkspacedValueWithContext(cueCtx, paths, runtimePrelude, homeLayers, preLayers, postWithFiles)
}

func compileWorkspacedValueWithContext(ctx *cue.Context, paths []string, runtimePrelude string, homeLayers bool, preLayers []compiledLayer, postLayers []compiledLayer) (cue.Value, error) {
	schemaBytes, err := schemaFS.ReadFile("schema.cue")
	if err != nil {
		return cue.Value{}, fmt.Errorf("read embedded cue schema: %w", err)
	}
	preludeCommonBytes, err := schemaFS.ReadFile("prelude_common.cue")
	if err != nil {
		return cue.Value{}, fmt.Errorf("read embedded cue prelude_common: %w", err)
	}
	preludeVariantFile := "prelude_codebase.cue"
	if homeLayers {
		preludeVariantFile = "prelude_home.cue"
	}
	preludeVariantBytes, err := schemaFS.ReadFile(preludeVariantFile)
	if err != nil {
		return cue.Value{}, fmt.Errorf("read embedded cue %s: %w", preludeVariantFile, err)
	}

	profileSource, err := fileProfileSource()
	if err != nil {
		return cue.Value{}, fmt.Errorf("mount file profiles: %w", err)
	}
	base := []namedSource{
		{Name: "schema.cue", Source: string(schemaBytes)},
		{Name: "compose-profiles.cue", Source: profileSource},
		{Name: "prelude_common.cue", Source: string(preludeCommonBytes)},
		{Name: preludeVariantFile, Source: string(preludeVariantBytes)},
		{Name: "runtime_prelude.cue", Source: runtimePrelude},
	}
	partial, err := buildPackage(ctx, base)
	if err != nil {
		return cue.Value{}, err
	}
	driverLayer, err := buildDriverWeightLayer(partial)
	if err != nil {
		return cue.Value{}, err
	}
	files := append([]namedSource{}, base...)
	if driverLayer != "" {
		files = append(files, namedSource{Name: "driver_weights.cue", Source: driverLayer})
	}
	for _, layer := range preLayers {
		files = append(files, namedSource{Name: layer.Name, Source: layer.Source})
	}
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return cue.Value{}, fmt.Errorf("read cue layer %s: %w", path, err)
		}
		files = append(files, namedSource{Name: path, Source: string(src)})
	}
	for _, layer := range postLayers {
		files = append(files, namedSource{Name: layer.Name, Source: layer.Source})
	}
	return buildPackage(ctx, files)
}

type namedSource struct {
	Name   string
	Source string
}

// buildPackage loads every layer as one CUE package so references such as
// runtime.home resolve across schema, preludes, and user files.
func buildPackage(ctx *cue.Context, files []namedSource) (cue.Value, error) {
	inst := &build.Instance{PkgName: "workspaced", User: true}
	for _, file := range files {
		syntax, err := parser.ParseFile(file.Name, file.Source, parser.ParseComments)
		if err != nil {
			return cue.Value{}, fmt.Errorf("parse cue layer %s: %w\n%s", file.Name, err, cueerrors.Details(err, nil))
		}
		if err := inst.AddSyntax(syntax); err != nil {
			return cue.Value{}, fmt.Errorf("add cue layer %s: %w", file.Name, err)
		}
	}
	v := ctx.BuildInstance(inst)
	if err := v.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("build cue config:\n%s", cueerrors.Details(err, nil))
	}
	return v, nil
}

type compiledLayer struct {
	Name   string
	Source string
}

func buildResolvedModuleLayers(configValue cue.Value, paths []string, discovered []Layer) ([]compiledLayer, []compiledLayer, error) {
	raw, err := decodeReadyMap(configValue)
	if err != nil {
		return nil, nil, fmt.Errorf("decode cue config before module config resolution: %w", err)
	}
	configJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("encode cue config before module config resolution: %w", err)
	}
	cfg, err := decodeConfig(configJSON)
	if err != nil {
		return nil, nil, err
	}

	modules, err := cfg.Modules()
	if err != nil {
		return nil, nil, fmt.Errorf("decode modules from config: %w", err)
	}
	modulesBaseDir := resolveModulesBaseDir(paths, discovered)
	if modulesBaseDir == "" {
		return nil, nil, nil
	}

	schemaByModule := map[string]string{}
	for modName, modEntry := range modules {
		modulePath, ok, err := resolveLocalModulePath(cfg, modName, modEntry, modulesBaseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve module %q for config resolution: %w", modName, err)
		}
		if !ok {
			continue
		}
		if !modulecue.Exists(modulePath) {
			continue
		}
		schemaText, err := modulecue.ConfigSyntaxWithRoot(modulePath, raw)
		if err != nil {
			return nil, nil, err
		}
		schemaByModule[modName] = schemaText
	}

	preLayers := make([]compiledLayer, 0, 1)
	if len(schemaByModule) > 0 {
		schemaLayer, err := buildModuleSchemaLayer(schemaByModule)
		if err != nil {
			return nil, nil, err
		}
		preLayers = append(preLayers, compiledLayer{
			Name:   "module_schemas.cue",
			Source: schemaLayer,
		})
	}
	postLayers := make([]compiledLayer, 0, 1)
	if hasDerivedDesktopModules(raw) {
		postLayers = append(postLayers, compiledLayer{
			Name:   "derived_modules.cue",
			Source: buildDerivedModulePrelude(),
		})
	}
	return preLayers, postLayers, nil
}

func buildModuleFileLayers(configValue cue.Value, paths []string, discovered []Layer) ([]compiledLayer, error) {
	if !configValue.Exists() {
		return nil, nil
	}
	raw, err := decodeReadyMap(configValue)
	if err != nil {
		return nil, fmt.Errorf("decode cue config before module file resolution: %w", err)
	}
	configJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode cue config before module file resolution: %w", err)
	}
	cfg, err := decodeConfig(configJSON)
	if err != nil {
		return nil, err
	}
	modules, err := cfg.Modules()
	if err != nil {
		return nil, fmt.Errorf("decode modules for file resolution: %w", err)
	}
	modulesBaseDir := resolveModulesBaseDir(paths, discovered)
	if modulesBaseDir == "" {
		return nil, nil
	}
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	layers := make([]compiledLayer, 0)
	for _, name := range names {
		entry := modules[name]
		if !entry.Enable {
			continue
		}
		modulePath, ok, err := resolveLocalModulePath(cfg, name, entry, modulesBaseDir)
		if err != nil {
			return nil, fmt.Errorf("resolve module %q for file: %w", name, err)
		}
		if !ok || !modulecue.Exists(modulePath) {
			continue
		}
		fileJSON, ok, err := modulecue.EvalFile(modulePath, raw)
		if err != nil {
			return nil, fmt.Errorf("module %q file: %w", name, err)
		}
		if !ok {
			continue
		}
		layers = append(layers, compiledLayer{
			Name:   "module_file_" + name + ".cue",
			Source: "package workspaced\nfile: " + string(fileJSON) + "\n",
		})
	}
	return layers, nil
}

func resolveModulesBaseDir(paths []string, discovered []Layer) string {
	for _, layer := range discovered {
		if layer.Name == "repo" || layer.Name == "dotfiles" {
			return filepath.Join(filepath.Dir(layer.Path), "modules")
		}
	}
	if len(paths) > 0 {
		return filepath.Join(filepath.Dir(paths[0]), "modules")
	}
	return ""
}

func hasDerivedDesktopModules(raw map[string]any) bool {
	modules, _ := raw["modules"].(map[string]any)
	if len(modules) == 0 {
		return false
	}
	_, hasBase16 := modules["base16"]
	_, hasGTK := modules["base16-gtk"]
	return hasBase16 || hasGTK
}

func buildDerivedModulePrelude() string {
	return `package workspaced

desktop: {
	dark_mode: *modules.base16.config.dark_mode | bool
	raw: {
		dconf: *modules["base16-gtk"].config.dconf | {
			[string]: [string]: _
		}
	}
}
`
}

func buildModuleSchemaLayer(schemaByModule map[string]string) (string, error) {
	moduleNames := make([]string, 0, len(schemaByModule))
	for name := range schemaByModule {
		moduleNames = append(moduleNames, name)
	}
	sort.Strings(moduleNames)

	moduleFields := make([]ast.Decl, 0, len(moduleNames))
	for _, name := range moduleNames {
		// Module schemas are package module and say workspaced.<field>.
		// Pasted into this package, that root is the top-level field.
		schemaText := strings.ReplaceAll(strings.TrimSpace(schemaByModule[name]), "workspaced.", "")
		expr, err := parser.ParseExpr(name+".module_config.cue", schemaText)
		if err != nil {
			return "", fmt.Errorf("parse module config schema for %q: %w", name, err)
		}
		moduleFields = append(moduleFields, &ast.Field{
			Label: ast.NewString(name),
			Value: &ast.StructLit{
				Elts: []ast.Decl{
					&ast.Field{
						Label: ast.NewIdent("config"),
						Value: expr,
					},
				},
			},
		})
	}

	file := &ast.File{
		Decls: []ast.Decl{
			&ast.Package{Name: ast.NewIdent("workspaced")},
			&ast.Field{
				Label: ast.NewIdent("modules"),
				Value: &ast.StructLit{
					Elts: moduleFields,
				},
			},
		},
	}
	formatted, err := format.Node(file)
	if err != nil {
		return "", fmt.Errorf("format generated module schema layer: %w", err)
	}
	return string(formatted), nil
}

func buildDriverWeightLayer(current cue.Value) (string, error) {
	shape := driver.RegisteredWeightShape()
	if len(shape) == 0 {
		return "", nil
	}

	ifaceNames := make([]string, 0, len(shape))
	for name := range shape {
		ifaceNames = append(ifaceNames, name)
	}
	sort.Strings(ifaceNames)

	driverFields := make([]ast.Decl, 0, len(ifaceNames))
	for _, ifaceName := range ifaceNames {
		providerIDs := shape[ifaceName]
		providerFields := make([]ast.Decl, 0, len(providerIDs))
		for _, providerID := range providerIDs {
			if hasDriverWeight(current, ifaceName, providerID) {
				continue
			}
			providerFields = append(providerFields, &ast.Field{
				Label: ast.NewString(providerID),
				Value: ast.NewBinExpr(
					token.OR,
					&ast.UnaryExpr{
						Op: token.MUL,
						X:  ast.NewLit(token.INT, strconv.Itoa(50)),
					},
					ast.NewIdent("int"),
				),
			})
		}
		if len(providerFields) == 0 {
			continue
		}
		driverFields = append(driverFields, &ast.Field{
			Label: ast.NewString(ifaceName),
			Value: &ast.StructLit{Elts: providerFields},
		})
	}
	if len(driverFields) == 0 {
		return "", nil
	}

	file := &ast.File{
		Decls: []ast.Decl{
			&ast.Package{Name: ast.NewIdent("workspaced")},
			&ast.Field{
				Label: ast.NewIdent("drivers"),
				Value: &ast.StructLit{Elts: driverFields},
			},
		},
	}
	formatted, err := format.Node(file)
	if err != nil {
		return "", fmt.Errorf("format generated driver weight layer: %w", err)
	}
	return string(formatted), nil
}

func hasDriverWeight(v cue.Value, ifaceName string, providerID string) bool {
	path := cue.MakePath(
		cue.Str("drivers"),
		cue.Str(ifaceName),
		cue.Str(providerID),
	)
	current := v.LookupPath(path)
	return current.Exists() && current.Err() == nil
}

func resolveLocalModulePath(cfg *Config, moduleName string, modEntry ModuleEntry, modulesBaseDir string) (string, bool, error) {
	workspaceRoot := filepath.Dir(modulesBaseDir)
	modulePath := strings.Trim(strings.TrimSpace(modEntry.Path), "/")
	if modulePath == "" {
		modulePath = filepath.ToSlash(filepath.Join("modules", moduleName))
	}

	if from := strings.TrimSpace(modEntry.From); from != "" {
		return resolveLocalSourceSpec(workspaceRoot, modulePath, from)
	}

	inputName := strings.TrimSpace(modEntry.Input)
	if inputName == "" {
		inputName = "self"
	}
	if strings.Contains(inputName, ":") {
		return resolveLocalSourceSpec(workspaceRoot, modulePath, inputName)
	}
	if inputName == "self" {
		return filepath.Join(workspaceRoot, filepath.FromSlash(modulePath)), true, nil
	}

	inputs, err := cfg.Inputs()
	if err != nil {
		return "", false, err
	}
	input, ok := inputs[inputName]
	if !ok {
		return "", false, nil
	}
	return resolveLocalSourceSpec(workspaceRoot, modulePath, input.From)
}

func resolveLocalSourceSpec(workspaceRoot, modulePath, spec string) (string, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "self" {
		return filepath.Join(workspaceRoot, filepath.FromSlash(modulePath)), true, nil
	}
	if !strings.HasPrefix(spec, "self:") {
		return "", false, nil
	}
	ref := strings.TrimSpace(strings.TrimPrefix(spec, "self:"))
	if ref == "" {
		return filepath.Join(workspaceRoot, filepath.FromSlash(modulePath)), true, nil
	}
	if modulePath != "" && modulePath != filepath.ToSlash(filepath.Join("modules", filepath.Base(modulePath))) {
		ref = strings.TrimRight(ref, "/") + "/" + strings.TrimLeft(modulePath, "/")
	}
	return filepath.Join(workspaceRoot, filepath.FromSlash(ref)), true, nil
}

func marshalWorkspacedValue(ctx context.Context, configValue cue.Value, paths []string, discovered []Layer) ([]byte, error) {
	if !configValue.Exists() {
		if len(discovered) > 0 {
			logger := logging.GetLogger(ctx)
			logger.Warn("experimental cue export produced empty result", "reason", "missing config", "layers", discovered)
		} else if len(paths) > 0 {
			logger := logging.GetLogger(ctx)
			logger.Warn("experimental cue export produced empty result", "reason", "missing config", "paths", paths)
		}
		return json.Marshal(map[string]any{})
	}
	b, err := configValue.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal cue config to json: %w", err)
	}
	if string(b) == "{}" && len(discovered) > 0 {
		logger := logging.GetLogger(ctx)
		logger.Warn("experimental cue export produced empty result", "reason", "config resolved to empty object", "layers", discovered)
	} else if string(b) == "{}" && len(paths) > 0 {
		logger := logging.GetLogger(ctx)
		logger.Warn("experimental cue export produced empty result", "reason", "config resolved to empty object", "paths", paths)
	}
	return b, nil
}

func formatWorkspacedValue(ctx context.Context, configValue cue.Value, paths []string, discovered []Layer) ([]byte, error) {
	return formatWorkspacedSyntax(ctx, configValue, paths, discovered, "export", "config",
		cue.Concrete(false),
		cue.Final(),
		cue.Definitions(false),
		cue.Hidden(false),
		cue.Optional(false),
		cue.Attributes(false),
		cue.Docs(false),
	)
}

func formatWorkspacedDef(ctx context.Context, configValue cue.Value, paths []string, discovered []Layer) ([]byte, error) {
	return formatWorkspacedSyntax(ctx, configValue, paths, discovered, "def", "def",
		cue.Concrete(false),
		cue.Definitions(true),
		cue.Hidden(false),
		cue.Optional(true),
		cue.Attributes(true),
		cue.Docs(true),
	)
}

// formatWorkspacedSyntax formats the config CUE value, or warns and
// returns "{}" when the value is missing. kind labels the empty-result warn
// ("export" / "def"); errLabel is used in format errors ("config" / "def").
func formatWorkspacedSyntax(ctx context.Context, configValue cue.Value, paths []string, discovered []Layer, kind, errLabel string, opts ...cue.Option) ([]byte, error) {
	if !configValue.Exists() {
		if len(discovered) > 0 {
			logger := logging.GetLogger(ctx)
			logger.Warn("experimental cue "+kind+" produced empty result", "reason", "missing config", "layers", discovered)
		} else if len(paths) > 0 {
			logger := logging.GetLogger(ctx)
			logger.Warn("experimental cue "+kind+" produced empty result", "reason", "missing config", "paths", paths)
		}
		return []byte("{}\n"), nil
	}

	n := configValue.Syntax(opts...)
	out, err := format.Node(n, format.Simplify())
	if err != nil {
		return nil, fmt.Errorf("format cue %s: %w", errLabel, err)
	}
	return append(out, '\n'), nil
}

func findUp(ctx context.Context, start string, name string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	// Determine the git root of the starting point (if any). We will not
	// walk above it when looking for workspaced.cue. This ensures nested
	// git repos don't see outer workspaced.cue files.
	gitRoot, gitErr := git.GetRoot(ctx, dir)
	hasGitBoundary := gitErr == nil && gitRoot != ""
	var absGit string
	if hasGitBoundary {
		var absErr error
		absGit, absErr = filepath.Abs(gitRoot)
		if absErr != nil {
			return "", absErr
		}
		absGit = filepath.Clean(absGit)
	}

	for {
		candidate := filepath.Join(dir, name)
		if fileExists(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		if hasGitBoundary {
			// dir started absolute and only walks via filepath.Dir, so parent is absolute.
			absParent := filepath.Clean(parent)
			if absParent != absGit && !strings.HasPrefix(absParent, absGit+string(filepath.Separator)) {
				// would leave the git repo root; stop without considering parent
				return "", nil
			}
		}
		dir = parent
	}
}

func ResolveWorkspaceCuePath(ctx context.Context, start string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	// Walk up from the starting directory to find the *closest* workspaced.cue,
	// but stop at the git root of the starting dir. This supports sub-workspaces
	// (cues deeper in the tree) inside a git repo, while ensuring that a git repo
	// nested inside another git repo does not inherit the parent's workspaced.cue.
	return findUp(ctx, start, "workspaced.cue")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func buildRuntimePrelude(ctx context.Context, resolvedInputs map[string]map[string]any, mode string) (string, error) {
	home, err := envdriver.GetHomeDir(ctx)
	if err != nil {
		// Fallback when drivers are not ready (early bootstrap).
		home, err = envdriver.ResolveHomeDir()
		if err != nil {
			return "", fmt.Errorf("user home dir: %w", err)
		}
	}
	configDir, err := envdriver.GetConfigDir(ctx)
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	userDataDir, err := envdriver.GetUserDataDir(ctx)
	if err != nil {
		return "", fmt.Errorf("user data dir: %w", err)
	}
	hostname, err := envdriver.GetHostname(ctx)
	if err != nil {
		return "", fmt.Errorf("hostname: %w", err)
	}

	if mode == "" {
		mode = filespine.ModeHome
	}
	runtimeMap := map[string]any{
		"is_phone":      envdriver.IsPhone(ctx),
		"hostname":      hostname,
		"home":          home,
		"config_dir":    configDir,
		"user_data_dir": userDataDir,
		"cpus":          runtime.NumCPU(),
		"goos":          runtime.GOOS,
		"goarch":        runtime.GOARCH,
		"memory":        memory.TotalMemory(),
		"mode":          mode,
	}
	// Optional: personal tree may be absent (CI, plain codebase checkouts).
	// Home prelude uses it via optional schema field; missing key fails only if cue requires it.
	if dotfilesRoot, err := envdriver.GetDotfilesRoot(ctx); err == nil && dotfilesRoot != "" {
		runtimeMap["dotfiles_root"] = dotfilesRoot
	}
	if len(resolvedInputs) > 0 {
		runtimeMap["inputs"] = resolvedInputs
	}

	b, err := json.Marshal(runtimeMap)
	if err != nil {
		return "", fmt.Errorf("marshal runtime cue prelude: %w", err)
	}
	return "package workspaced\n\nruntime: " + string(b) + "\n", nil
}

// decodeReadyMap is a partial JSON view of v. Incomplete fields (for example
// file before a module supplies type) are omitted so input and
// module resolution can run before the last unify.
func decodeReadyMap(v cue.Value) (map[string]any, error) {
	decoded, ok, err := decodeReadyValue(v)
	if err != nil {
		return nil, err
	}
	if !ok || decoded == nil {
		return map[string]any{}, nil
	}
	m, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cue value is not a struct")
	}
	return m, nil
}

func decodeReadyValue(v cue.Value) (any, bool, error) {
	if !v.Exists() {
		return nil, false, nil
	}
	if err := v.Err(); err != nil {
		return nil, false, err
	}
	if raw, err := v.MarshalJSON(); err == nil {
		var out any
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, false, err
		}
		return out, true, nil
	}
	if v.IncompleteKind() != cue.StructKind {
		return nil, false, nil
	}
	iter, err := v.Fields()
	if err != nil {
		return nil, false, nil
	}
	out := map[string]any{}
	for iter.Next() {
		child, ok, err := decodeReadyValue(iter.Value())
		if err != nil {
			return nil, false, err
		}
		if !ok {
			continue
		}
		out[iter.Selector().Unquoted()] = child
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func resolveRuntimeInputs(configValue cue.Value, paths []string, discovered []Layer) (map[string]map[string]any, error) {
	raw, err := decodeReadyMap(configValue)
	if err != nil {
		return nil, fmt.Errorf("decode cue config for input resolution: %w", err)
	}
	inputsRaw, _ := raw["inputs"].(map[string]any)
	if len(inputsRaw) == 0 {
		return nil, nil
	}

	type inputCfg struct {
		From    string `json:"from"`
		Version string `json:"version"`
	}
	cfgInputs := map[string]inputCfg{}
	tmp, err := json.Marshal(inputsRaw)
	if err != nil {
		return nil, fmt.Errorf("marshal inputs for resolution: %w", err)
	}
	if err := json.Unmarshal(tmp, &cfgInputs); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeInputs, err)
	}

	modulesBaseDir := resolvedSelfModulesBase(paths, discovered)
	workspaceRoot := filepath.Dir(modulesBaseDir)
	home, err := envdriver.ResolveHomeDir()
	if err != nil {
		return nil, fmt.Errorf("user home dir: %w", err)
	}
	out := map[string]map[string]any{}
	for name, input := range cfgInputs {
		spec := strings.TrimSpace(input.From)
		if spec == "" {
			continue
		}
		if spec == "self" || name == "self" {
			out[name] = map[string]any{"path": workspaceRoot}
			continue
		}

		provider, target, ok := parseInputSpec(spec)
		if !ok {
			return nil, fmt.Errorf("resolve runtime input %q: %w: %q", name, ErrInvalidInputSpec, spec)
		}
		switch provider {
		case "github":
			cacheKey := githubCacheKey(target, input.Version)
			out[name] = map[string]any{
				"path": filepath.Join(home, ".cache", "workspaced", "sources", "github", hashPath(cacheKey)),
			}
		case "local":
			base := target
			if !filepath.IsAbs(base) {
				base = filepath.Join(workspaceRoot, base)
			}
			out[name] = map[string]any{"path": filepath.Clean(base)}
		default:
			out[name] = map[string]any{
				"provider": provider,
				"target":   target,
			}
		}
	}
	return out, nil
}

func resolvedSelfModulesBase(paths []string, discovered []Layer) string {
	for _, layer := range discovered {
		if layer.Name == "repo" || layer.Name == "dotfiles" {
			return filepath.Join(filepath.Dir(layer.Path), "modules")
		}
	}
	for _, path := range paths {
		return filepath.Join(filepath.Dir(path), "modules")
	}
	return filepath.Join(".", "modules")
}

func parseInputSpec(spec string) (string, string, bool) {
	provider, target, ok := strings.Cut(strings.TrimSpace(spec), ":")
	if !ok {
		return "", "", false
	}
	provider = strings.TrimSpace(provider)
	target = strings.TrimSpace(target)
	if provider == "" || target == "" {
		return "", "", false
	}
	return provider, target, true
}

func githubCacheKey(repo, version string) string {
	ref := strings.TrimSpace(version)
	if ref == "" {
		ref = "HEAD"
	}
	return "v4:repo:" + strings.Trim(strings.TrimSpace(repo), "/") + "@" + ref
}

func hashPath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
