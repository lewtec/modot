package configcue

import (
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

	t.Run("core:place is rewritten onto file.home and disabled", func(t *testing.T) {
		t.Parallel()
		u := cueCtx.CompileString(`
package modot
modules: best_practices: {
	from: "core:place"
	config: {
		items: {"skills/bp/go": "/tmp/go"}
		steps: {
			"10_require": {op: "require", patterns: {skill: "SKILL.md"}}
		}
	}
}
`, cue.Filename("user.cue"))
		require.NoError(t, u.Err())
		v := schema.Unify(u)
		require.NoError(t, v.Err())
		enabled, err := v.LookupPath(cue.ParsePath("modules.best_practices.enable")).Bool()
		require.NoError(t, err)
		require.False(t, enabled)
		src, err := v.LookupPath(cue.ParsePath(`file.home."skills/bp/go".source.best_practices`)).String()
		require.NoError(t, err)
		require.Equal(t, "/tmp/go", src)
		op, err := v.LookupPath(cue.ParsePath(`file.home."skills/bp/go".steps."10_require".op`)).String()
		require.NoError(t, err)
		require.Equal(t, "require", op)
	})

	cases := []struct {
		name    string
		cue     string
		wantErr bool
	}{
		{
			name: "accepts move and require steps",
			cue: `
package modot
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
`,
		},
		{
			name:    "rejects unknown step op",
			wantErr: true,
			cue: `
package modot
modules: best_practices: {
	from: "core:place"
	config: {
		steps: {x: {op: "reject"}}
	}
}
`,
		},
		{
			name:    "rejects move without from",
			wantErr: true,
			cue: `
package modot
modules: best_practices: {
	from: "core:place"
	config: {
		steps: {x: {op: "move", to: "entry.md"}}
	}
}
`,
		},
		{
			name: "non-place module config stays open",
			cue: `
package modot
modules: other: {
	from: "self"
	config: {anything: true, nested: {x: 1}}
}
`,
		},
		{
			name: "module without from field",
			cue: `
package modot
modules: fontconfig: {
	input: "self"
	path:  "fontconfig"
	config: {enable: true}
}
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := unifyUser(t, tc.cue)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
