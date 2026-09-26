package configcue

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/lewtec/lewkit/x/fs/compose"
	"github.com/lewtec/modot/internal/filespine"
	"github.com/lewtec/modot/internal/placestep"
)

var (
	errNilCueContext = errors.New("nil cue context")
	errEmptyMountSrc = errors.New("empty src")
)

// fileProfileSource installs the compose schema and closes file to the
// profile names. A flat key is a schema error.
// A profile entry is a compose file or a mount of another tree.
// Mount lives on the profile, so codebase and system see it. core:place does not.
func fileProfileSource() (string, error) {
	// Hidden path: the schema is _compose. file.<profile> is not #Tree,
	// because a mount entry is not a compose file.
	source, err := compose.Mount("_fileSchema.tree")
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.WriteString("package modot\n")
	builder.WriteString(source)
	if !strings.HasSuffix(source, "\n") {
		builder.WriteByte('\n')
	}
	builder.WriteString(`#MountStepMove: close({
	op:   "move"
	from: string
	to:   string
})
#MountStepRequire: close({
	op: "require"
	patterns: [string]: string
})
#MountStep: #MountStepMove | #MountStepRequire
#FileMount: close({
	type: "mount"
	// One tree, or several trees merged under this path.
	src: string | {[string]: string}
	steps?: [string]: #MountStep
})
#Profile: [string]: _compose.#File | #FileMount
file: close({
`)
	for _, name := range filespine.Profiles {
		fmt.Fprintf(&builder, "\t%s?: #Profile\n", name)
	}
	builder.WriteString("})\n")
	return builder.String(), nil
}

// mountFileProfiles unifies fileProfileSource onto value.
func mountFileProfiles(value cue.Value) (cue.Value, error) {
	source, err := fileProfileSource()
	if err != nil {
		return cue.Value{}, err
	}

	cueContext := value.Context()
	if cueContext == nil {
		return cue.Value{}, fmt.Errorf("mount file profiles: %w", errNilCueContext)
	}
	layer := cueContext.CompileString(source, cue.Filename("compose-profiles.cue"))
	if err := layer.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("compile file profiles: %w", err)
	}
	out := value.Unify(layer)
	if err := out.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("unify file profiles: %w", err)
	}
	return out, nil
}

// FileProfiles parses each profile visible for the runtime mode.
// A missing profile is an empty tree. Flat keys fail before this runs.
func (c *Config) FileProfiles() (map[string]*compose.Tree, error) {
	if c == nil {
		return map[string]*compose.Tree{}, nil
	}
	mode := c.RuntimeMode()
	out := make(map[string]*compose.Tree)
	root := c.Cue()
	hasCue := root.Exists()
	var fileVal cue.Value
	if hasCue {
		fileVal = root.LookupPath(cue.ParsePath("file"))
	}
	for _, name := range filespine.Visible(mode) {
		if !hasCue {
			out[name] = compose.New()
			continue
		}
		parsed, _, err := parseProfile(fileVal.LookupPath(cue.ParsePath(name)))
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		out[name] = parsed
	}
	return out, nil
}

// Mount is one or more trees merged into a profile path. Steps are the place pipeline.
// Steps run on each source tree before the path prefix is added.
type Mount struct {
	Srcs  []string
	Steps map[string]placestep.Step
}

// FileMounts is dest prefix → mount for each visible profile.
// A source is a filesystem path or an input ref (alias:rel). The tree is
// merged into that profile by the file spine.
func (c *Config) FileMounts() (map[string]map[string]Mount, error) {
	out := map[string]map[string]Mount{}
	if c == nil {
		return out, nil
	}
	root := c.Cue()
	if !root.Exists() {
		return out, nil
	}
	fileVal := root.LookupPath(cue.ParsePath("file"))
	for _, name := range filespine.Visible(c.RuntimeMode()) {
		_, mounts, err := parseProfile(fileVal.LookupPath(cue.ParsePath(name)))
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		if len(mounts) > 0 {
			out[name] = mounts
		}
	}
	return out, nil
}

// parseProfile splits mount entries out of a profile. The rest is a compose tree.
// A missing profile is an empty tree.
func parseProfile(value cue.Value) (*compose.Tree, map[string]Mount, error) {
	if !value.Exists() {
		return compose.New(), nil, nil
	}
	if err := value.Err(); err != nil {
		return nil, nil, err
	}
	iterator, err := value.Fields()
	if err != nil {
		return nil, nil, err
	}
	mounts := map[string]Mount{}
	cueContext := value.Context()
	files := cueContext.CompileString("{}", cue.Filename("profile-files.cue"))
	if err := files.Err(); err != nil {
		return nil, nil, err
	}
	for iterator.Next() {
		name := iterator.Selector().Unquoted()
		field := iterator.Value()
		typeValue := field.LookupPath(cue.ParsePath("type"))
		kind := ""
		if typeValue.Exists() {
			var err error
			kind, err = typeValue.String()
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", name, err)
			}
		}
		if kind == "mount" {
			mount, err := decodeMount(field)
			if err != nil {
				return nil, nil, fmt.Errorf("mount %s: %w", name, err)
			}
			mounts[name] = mount
			continue
		}
		files = files.FillPath(cue.ParsePath(strconv.Quote(name)), field)
		if err := files.Err(); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	tree, err := compose.Parse(files)
	if err != nil {
		return nil, nil, err
	}
	return tree, mounts, nil
}

func decodeMount(field cue.Value) (Mount, error) {
	srcs, err := decodeMountSrcs(field.LookupPath(cue.ParsePath("src")))
	if err != nil {
		return Mount{}, err
	}
	stepsValue := field.LookupPath(cue.ParsePath("steps"))
	if !stepsValue.Exists() {
		return Mount{Srcs: srcs}, nil
	}
	iterator, err := stepsValue.Fields()
	if err != nil {
		return Mount{}, err
	}
	steps := map[string]placestep.Step{}
	for iterator.Next() {
		var decoded struct {
			Op       string            `json:"op"`
			From     string            `json:"from"`
			To       string            `json:"to"`
			Patterns map[string]string `json:"patterns"`
		}
		if err := iterator.Value().Decode(&decoded); err != nil {
			return Mount{}, fmt.Errorf("step %s: %w", iterator.Selector().Unquoted(), err)
		}
		steps[iterator.Selector().Unquoted()] = placestep.Step{
			Op:       decoded.Op,
			From:     decoded.From,
			To:       decoded.To,
			Patterns: decoded.Patterns,
		}
	}
	return Mount{Srcs: srcs, Steps: steps}, nil
}

func decodeMountSrcs(value cue.Value) ([]string, error) {
	if !value.Exists() {
		return nil, errEmptyMountSrc
	}
	if src, err := value.String(); err == nil {
		src = strings.TrimSpace(src)
		if src == "" {
			return nil, errEmptyMountSrc
		}
		return []string{src}, nil
	}
	iterator, err := value.Fields()
	if err != nil {
		return nil, err
	}
	var srcs []string
	for iterator.Next() {
		src, err := iterator.Value().String()
		if err != nil {
			return nil, fmt.Errorf("src.%s: %w", iterator.Selector().Unquoted(), err)
		}
		src = strings.TrimSpace(src)
		if src == "" {
			return nil, fmt.Errorf("src.%s: %w", iterator.Selector().Unquoted(), errEmptyMountSrc)
		}
		srcs = append(srcs, src)
	}
	if len(srcs) == 0 {
		return nil, errEmptyMountSrc
	}
	return srcs, nil
}
