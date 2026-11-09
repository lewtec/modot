package configcue

import (
	"strings"

	"cuelang.org/go/cue"
)

// Provenance names the file that supplied a merged key.
//
// CUE records the source file of the last conjunct that set a value, so this is
// the file that won, not every file that mentioned the key. The build names in
// compileModotValueWithContext are the entry and module file paths the loader
// joined, which is exactly what a reader wants to see.
type Provenance struct {
	// Files maps a dotted key path to the file that last set it.
	Files map[string]string `json:"files"`
	// Builtin maps a path whose winning value came from an embedded layer
	// (schema, prelude, runtime) to that layer's file name. Such a value is a
	// real default, which is not the same as a value CUE computed.
	Builtin map[string]string `json:"builtin"`
}

// Provenance walks a Configuration and records the source file of every value.
func (r EvaluationResult) Provenance() Provenance {
	return provenanceOf(r.Value)
}

func provenanceOf(value cue.Value) Provenance {
	out := Provenance{Files: map[string]string{}, Builtin: map[string]string{}}
	if !value.Exists() {
		return out
	}
	value.Walk(func(val cue.Value) bool {
		if val.Err() != nil {
			return true
		}
		source := val.Source()
		if source == nil {
			return true
		}
		file := source.Pos().Filename()
		path := val.Path().String()
		if path == "" || file == "" {
			return true
		}
		if isUserFile(file) {
			out.Files[path] = file

			return true
		}
		// An embedded layer: keep the name so the dashboard can say a value is
		// a default instead of leaving it unexplained.
		out.Builtin[path] = file

		return true
	}, nil)
	return out
}

// isUserFile reports whether name is an entry or module file the loader read
// from disk. Embedded sources keep the bare build name
// compileModotValueWithContext gives them, so a path separator tells them apart.
func isUserFile(name string) bool {
	if name == "" {
		return false
	}
	return strings.ContainsAny(name, `/\`)
}
