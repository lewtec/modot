# File spine

`workspaced.file` is a closed map of profiles. Each profile is one compose tree from `github.com/lewtec/lewkit/x/fs/compose`. A flat key under `workspaced.file` is a schema error.

Profiles are `home`, `codebase`, `etc`, `usr`, `root`, `var`, `bin`, and `system`.

`pkg/filespine` selects the profiles a runtime mode emits (`NamespaceVisible`) and the paths the template expander renders (`IsTemplatePath`). `internal/configcue` mounts the closed map with `mountFileProfiles`.

## Mode

`workspaced.runtime.mode` is `home`, `codebase`, or `system`. A missing mode is `home`.

| Mode | Profiles |
|---|---|
| `home` | `home` |
| `codebase` | `codebase` |
| `system` | `etc`, `usr`, `root`, `var`, `bin`, `system` |

`home`, `codebase`, and `system` use the apply target directory. The other profiles use a fixed directory: `etc` is `/etc`, `usr` is `/usr`, `root` is `/`, `var` is `/var`, and `bin` is `/usr/local/bin`.

`Open(name)` on a profile filesystem returns the combined file. `name` is an `fs.FS` path. It has no leading `/`, no `~`, and no `..`.

## File types

| `type` | `values` | Encode |
|---|---|---|
| `lines` | any number of slots | sort the keys, join with a newline |
| `text` | one slot | that slot |
| `ref` | one ref slot | bytes from the profile filesystem |
| `json`, `toml`, `yaml`, `ini`, `xml` | one map | marshal that map |

A second type on one path is an error. The same slot key with a different body is an error. A ref is a relative name in the profile filesystem. An absolute ref is an error.

```cue
#Slot: string | close({kind: "text", text: string}) | close({kind: "ref", ref: string})
```

A bare string is a text slot. There is no `{kind: "env"}`.

## Module file

`module.file` in `module.cue` uses the same profile map. Enabled modules contribute. The host value and the module value unify in CUE. Parse runs once after that unify.

```cue
module: file: home: ".config/foo.toml": {
	type: "toml"
	values: {theme: "base16"}
}
```

`module.file` can read `workspaced.*` (runtime, other module config).

## Lowering

Render templates first. `IsTemplatePath` is true when any suffix is `.tmpl`, so `file.tmpl.sh` is a template. Compose squash only checks the last suffix.

Squash each visible profile, then merge that tree into the parsed profile.

| Source under the profile | Result |
|---|---|
| `.bashrc.d.tmpl/20-alias.sh` | `lines` slot `20-alias.sh` on `.bashrc` |
| `.bashrc.tmpl` | one file `.bashrc` |
| `.gitconfig` | `ref` opened from the profile filesystem |

A directory under `.d.tmpl` is an error. A symlink in the profile filesystem stays a symlink at the apply target. Plan text uses the scanner's module name and relative path.

## Example

```cue
workspaced: file: home: ".bashrc": {
	type: "lines"
	values: {
		"00-umask": "umask 022"
		"10-path":  "export PATH=$HOME/bin:$PATH"
	}
}

workspaced: file: home: ".config/foo.json": {
	type: "json"
	values: {
		port: 8080
		name: "foo"
	}
}
```

## Out of scope

- writable dest filesystem
- `runtime.env`
