package configcue

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestPlaceModuleConfigSchema(t *testing.T) {
	t.Parallel()

	schemaBytes, err := schemaFS.ReadFile("schema.cue")
	require.NoError(t, err)
	cueCtx := cuecontext.New()
	schema := cueCtx.CompileBytes(schemaBytes, cue.Filename("schema.cue"))
	require.NoError(t, schema.Err(), "schema")

	unifyUser := func(t *testing.T, user string) error {
		t.Helper()
		u := cueCtx.CompileString(user, cue.Filename("user.cue"))
		if err := u.Err(); err != nil {
			return err
		}
		v := schema.Unify(u)
		if err := v.Err(); err != nil {
			return err
		}
		// Same hardness as export: concrete JSON of modules.
		mod := v.LookupPath(cue.ParsePath("modules"))
		if err := mod.Err(); err != nil {
			return err
		}
		_, err := mod.MarshalJSON()
		return err
	}

	t.Run("accepts move and require steps", func(t *testing.T) {
		t.Parallel()
		err := unifyUser(t, `
package workspaced
modules: best_practices: {
	from: "core:place"
	config: {
		items: {"skills/bp/go": "/tmp/go"}
		steps: {
			"10_require": {op: "require", patterns: {skill: "SKILL.md"}}
			"20_demote":  {op: "move", from: "SKILL.md", to: "entry.md"}
		}
		topics: {go: true}
	}
}
`)
		require.NoError(t, err, "unify")
	})

	t.Run("rejects unknown step op", func(t *testing.T) {
		t.Parallel()
		err := unifyUser(t, `
package workspaced
modules: best_practices: {
	from: "core:place"
	config: {
		steps: {x: {op: "reject"}}
	}
}
`)
		require.Error(t, err, "expected schema error for unknown op")
		msg := err.Error()
		require.True(t,
			strings.Contains(msg, "disjunction") || strings.Contains(msg, "reject") || strings.Contains(msg, "op"),
			"unexpected error: %v", err)
	})

	t.Run("rejects move without from", func(t *testing.T) {
		t.Parallel()
		err := unifyUser(t, `
package workspaced
modules: best_practices: {
	from: "core:place"
	config: {
		steps: {x: {op: "move", to: "entry.md"}}
	}
}
`)
		require.Error(t, err, "expected schema error for incomplete move")
	})

	t.Run("non-place module config stays open", func(t *testing.T) {
		t.Parallel()
		err := unifyUser(t, `
package workspaced
modules: other: {
	from: "self"
	config: {anything: true, nested: {x: 1}}
}
`)
		require.NoError(t, err, "unify")
	})

	t.Run("module without from field (input only)", func(t *testing.T) {
		t.Parallel()
		// Regression: if from == "core:place" must not require optional from.
		err := unifyUser(t, `
package workspaced
modules: fontconfig: {
	input: "self"
	path:  "fontconfig"
	config: {enable: true}
}
`)
		require.NoError(t, err, "unify")
	})
}
