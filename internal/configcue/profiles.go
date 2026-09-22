package configcue

import (
	"errors"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"github.com/lewtec/lewkit/x/fs/compose"
	"github.com/lucasew/workspaced/pkg/filespine"
)

var errNilCueContext = errors.New("nil cue context")

// mountFileProfiles installs the compose schema and closes workspaced.file
// to the profile names. A flat key is a schema error.
func mountFileProfiles(v cue.Value) (cue.Value, error) {
	source, err := compose.Mount("workspaced.file.home")
	if err != nil {
		return cue.Value{}, err
	}
	var b strings.Builder
	b.WriteString(source)
	if !strings.HasSuffix(source, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("workspaced: file: close({\n")
	for _, name := range filespine.Profiles {
		fmt.Fprintf(&b, "\t%s?: _compose.#Tree\n", name)
	}
	b.WriteString("})\n")

	ctx := v.Context()
	if ctx == nil {
		return cue.Value{}, fmt.Errorf("mount file profiles: %w", errNilCueContext)
	}
	layer := ctx.CompileString(b.String(), cue.Filename("compose-profiles.cue"))
	if err := layer.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("compile file profiles: %w", err)
	}
	out := v.Unify(layer)
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
		parsed, err := compose.Parse(fileVal.LookupPath(cue.ParsePath(name)))
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		out[name] = parsed
	}
	return out, nil
}
