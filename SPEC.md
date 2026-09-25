# Modot specification

This document is the law for the modot program: how one entry file composes modules into dotfiles.

Status: approved
Genre: cli

The key words MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY in this
document are to be interpreted as described in BCP 14 (RFC 2119,
RFC 8174) when, and only when, they appear in all capitals.

## Intention

Job: Modot builds dotfiles for programs by composing modules.

Non-goals:

1. A package manager. Tool install remains a way to fetch one program at a locked version.
2. A public Go API. Reusable code belongs in lewkit.
3. A second config language.
4. Discovery of modules that the entry file does not name.
5. A compatibility window for the workspaced filenames.

Inherited C (cite the file):

- Go 1.27 and CUE (`go.mod`, `cuelang.org/go`).
- lewkit packages already required by `go.mod`: `x/cmd`, `x/taskgroup`, `x/tool`, `x/logging`, `x/path`, `x/db`, `x/fs/compose`.
- Command history rows in sqlite (`internal/db/sqlite/migrations/0001_create_history_table.up.sql`).
- Profile trees `home`, `codebase`, and `system` (`internal/filespine`).

## Technique

| ID | Input | Rule | Output |
|----|-------|------|--------|
| TEC-01 | The entry `modot.cue` | Read `modules` from that file only. Each entry is `{input, path}`. `enable` defaults to true. `input: "self"` is the directory of the entry file. A disabled entry selects nothing. | The set of enabled module directories |
| TEC-02 | One enabled module directory | Evaluate that directory's `modot.cue` with the directory as the path base. Resolve each source path against that directory. A source path names a file to read. A key under `file.<profile>` is a destination path and stays unresolved here. | One fragment whose source paths are rooted at that module |
| TEC-03 | The entry file plus every TEC-02 fragment | Unify the fragments as one CUE value, then evaluate that value once. | One Configuration |
| TEC-04 | One module directory selected by TEC-01 | Walk template files only when that `modot.cue` is inside a git work tree and below the work-tree root. Paths are relative to that directory. The entry file at the work-tree root contributes no template files. A `modot.cue` outside a git work tree contributes no template files. | Template files for that module |
| TEC-05 | One Configuration | `runtime.mode` selects one profile: `home`, `codebase`, `system`. A missing mode selects `home`. `file.<profile>` is a closed map. A flat key under `file` is a failure. File types are `lines`, `text`, `ref`, `json`, `toml`, `yaml`, `ini`, `xml`. Two types on one destination path are a failure. | The Profile tree for that mode |
| TEC-06 | A command name from the command table | The command performs the transition in that table on the named type. | The type after the transition |
| TEC-07 | A finished command | Exit 0. Records go to stdout. | A successful run |
| TEC-08 | `--no-cache` and `MODOT_NO_CACHE` | The flag and the env var arm the same bit. The env value `0`, `false`, `no`, `off` leaves the bit clear. An empty env value leaves the bit clear. When the bit is set and `--dry-run` is clear, warm install, module, source, and shell caches miss, and a deploy noop becomes an update. | A cold run |
| TEC-09 | `lint` and `formatter` in the Configuration | Each tool is one `#CheckTool` value in CUE. The runner is generic. Codecs are a closed set. | Lint findings and formatted files |

## Tooling

| TEC | Tool | Relation | We do not | Cite |
|-----|------|----------|-----------|------|
| TEC-01 | `cuelang.org/go` | adopt | Write a CUE evaluator | `go.mod` |
| TEC-02 | `cuelang.org/go` | adopt | Resolve source paths in a private expression language | `go.mod` |
| TEC-03 | `cuelang.org/go` | implement | Import a lewkit loader. Lewkit does not join module files into one configuration. | `none`: `github.com/lewtec/lewkit` has no such loader. CUE use there is limited to `x/fs/compose/compose_test.go` |
| TEC-04 | `github.com/lewtec/lewkit/x/fs/compose` | adopt | Build a second file-tree merge | `internal/filespine` |
| TEC-05 | `github.com/lewtec/lewkit/x/fs/compose` | adopt | Store profile trees outside `file` | `internal/filespine` |
| TEC-06 | `github.com/lewtec/lewkit/x/cmd` | adopt | Build a second CLI framework | `cmd/modot/spec.go` |
| TEC-07 | `github.com/lewtec/lewkit/x/cmd` | adopt | Invent an exit-code scheme | `cmd/modot/spec.go` |
| TEC-08 | `github.com/lewtec/lewkit/x/cmd` | wrap | Read `MODOT_NO_CACHE` at each call site | `cmd/modot/spec.go`, `internal/cmdctx` |
| TEC-09 | `cuelang.org/go` | adopt | Ship one Go package per linter | ADR-0004 |

| Cell | Pick | Bound | Implements | Cite if C |
|------|------|--------|------------|-----------|
| Language | Go 1.27 and CUE | C | TEC-01, TEC-03 | `go.mod` |
| Runtime | The machine that runs the `modot` binary | C | TEC-06 | `dist/` |
| Persistence | `modot.lock.json` and the sqlite history file | C | TEC-06 | `internal/modfile`, `internal/db/db.go` |
| UI | CLI. Records on stdout. Context on stderr | C | TEC-07 | `cmd/modot/spec.go` |
| Packaging | The release layout in `dist/config.yaml` | C | TEC-06 | `dist/config.yaml` |
| Identity | None. One operator on one machine | D | TEC-07 | |
| Host OS | The GOOS values already shipped under `dist/` | C | TEC-06 | `dist/` |

A technique that already exists in lewkit MUST be called from lewkit. This repository MUST NOT copy that technique.

The module path is `github.com/lewtec/modot`. Importable packages stop at `internal/`.

## Terminology

| Concept | Approved | Banned |
|---------|----------|--------|
| This program | modot | workspaced, as a current name |
| Entry file | `modot.cue` | config file, flake |
| A directory named by `modules` that contains `modot.cue` | module | package, flake, plugin |
| The single evaluated CUE value | Configuration | config bag, module config |
| `modot.lock.json` | Lock | sum file |
| `file.home`, `file.codebase`, `file.system` | Profile | dest tree, namespace |
| A string that names a file to read from a module | source path | import |
| A key under `file.<profile>` | destination path | target, output path |
| `input: "self"` | self | repo root, as a synonym |
| One locked program in the tool store | Tool | package |
| One sqlite history row | History | log entry |
| A past decision recorded in this file | ADR | spec note |

The CUE `package` clause has no meaning. `package picuinha` and a missing clause are the same input.

## Types

| Command | Type it mutates | Transition | Bad input |
|---------|-----------------|------------|-----------|
| `home plan`, `codebase plan` | Configuration | Read the Profile. Write nothing | Load failure |
| `home apply`, `codebase apply`, `system` | Profile | Write the Profile for `runtime.mode` to `--prefix` | Load failure |
| `home config`, `codebase config` | Configuration | Read the Configuration. Write nothing | Load failure |
| `mod lock`, `mod tidy` | Lock | Write pins for the TEC-01 set | Load failure |
| `tool` | Tool | Install the locked program | Load failure. Missing pin |
| `driver`, `open` | none | Perform the live OS action | Driver failure |
| `init` | Configuration | Write a starter `modot.cue` | Starter write failure |
| `utils history` | History | Read rows | Store open failure |
| `is` | none | Print a detection record | Detection failure |
| `self-install`, `self-update` | Tool | Place this binary in the tool store | Install failure |
| `svc`, `daemon` | none | Run the background process | Start failure |
| `experiments`, `utils` other than `history` | none | Run the existing utility | Command failure |
| `home backup`, `home sync` | Profile | Run the existing backup action | Load failure |
| `codebase lint`, `codebase format` | Profile | Run TEC-09 on the codebase root | Load failure |

`--prefix` defaults to `~` for `home`, `.` for `codebase`, and `/` for `system`.

`--dry-run` performs the read steps and writes nothing.

### Stored entity

| Entity | Kind | Identity authority | A/B rels `(min,max)` | Root | Invariant IDs |
|--------|------|--------------------|----------------------|------|---------------|
| History | entity | sqlite row id | one command run produces one row `(1,1)` | yes | INV-08 |

| Rel | A role | B role | A (min,max) | B (min,max) | Identifying? | Owner | Link attrs | Ban |
|-----|--------|--------|-------------|-------------|--------------|-------|------------|-----|
| records | command run | History | (1,1) | (1,1) | yes | History row | command, cwd, timestamp, exit_code, duration_ms | a second log table |

Configuration, Module, Lock, Profile, and Tool are files. They are not sqlite entities. A Module's identity is `input` plus `path`. A Lock's identity is the path `modot.lock.json` beside the entry file. A Tool's identity is its lock pin.

## Invariants

| ID | Predicate | On | Forbidden bypass |
|----|-----------|----|------------------|
| INV-01 | The entry file is `modot.cue` | Configuration | Loading `workspaced.cue` |
| INV-02 | Every loaded module is named by the entry file's `modules` ref | Module | A directory walk that selects modules |
| INV-03 | A module file does not add modules | Module | Reading `modules` from a module file during TEC-01 |
| INV-04 | A source path resolves against that module directory | Module | Joining relative source paths before TEC-02 |
| INV-05 | A source path whose result leaves the module directory fails TEC-02 | Module | Accepting `..` that escapes the directory |
| INV-06 | The Lock records pins for the TEC-01 set | Lock | Hand-editing pins as the source of intent |
| INV-07 | Apply writes only the Profile selected by `runtime.mode` | Profile | Writing `file.home` during a codebase run |
| INV-08 | A History row stores command, cwd, timestamp, exit code, and duration | History | A row with a null command |
| INV-09 | This file is the only constitution | SPEC | A second `SPEC.md`. A `docs/specs/` file that wins over this file |
| INV-10 | No import path under `github.com/lewtec/modot` is a supported API except `cmd/modot` | repository | Keeping `pkg/` importable |

## Errors

One reaction for every row: the command blows up. Stderr names the entry file, the module directory, and the failing field, when those exist. Stdout is empty. The process exits non-zero. The target of the command is left as the failure found it.

| Public operation | Bad input | One reaction |
|------------------|-----------|--------------|
| Any command in the command table | The bad input named in that row | The reaction above |
| Load | `workspaced.cue`, `workspaced.lock.json`, a `WORKSPACED_*` variable | The reaction above. The message names the leftover and the modot spelling |
| Load | A CUE conflict, a missing module directory, a lock hash mismatch | The reaction above |
| TEC-02 | A source path that escapes the module directory | The reaction above |
| TEC-05 | Two types on one destination path. A flat `file` key. A directory under a `.d.tmpl` tree | The reaction above |

## Actors

N/A: genre=cli. The matrix omits actors.

## Capabilities

N/A: genre=cli. The matrix omits capabilities.

## Quality

| Concern | Measure. If it cannot happen, why |
|---------|----------------------------------|
| Exit contract | A finished command exits 0 and MAY write records to stdout. A command that blows up exits non-zero, writes the context to stderr, and writes nothing to stdout. |
| Untrusted input | The load reads the entry `modot.cue` and the module directories TEC-01 named. A remote module is read only after its Lock hash matches. No other path becomes a module. |

## Security

In scope: which files a run reads, and which Profile a run writes.

Why it cannot happen (if claiming none):

Residual risk: the entry file is the operator's own configuration. A module named by that file runs as the operator.

## Success

- [ ] A module directory that the entry `modules` ref does not name contributes no fields and no template files.
- [ ] Two module files that set the same concrete general field to different values make the command exit non-zero.
- [ ] A source path `theme.toml` inside module `modules/git` resolves under `modules/git`.
- [ ] The repo-root `modot.cue` contributes no template files from its subdirectories.
- [ ] `~/modot.cue` outside a git work tree contributes no template files.
- [ ] A directory that contains `workspaced.cue` makes the command exit non-zero.
- [ ] `package` in a loaded file does not change the Configuration.
- [ ] No package outside this repository can import `github.com/lewtec/modot/pkg/...`.

## Later work

None.

## Assumptions

| ID | Fact | If false |
|----|------|----------|
| AS-01 | The operator is the only user of this program (stated 2026-09-25) | A migration window for old filenames would be required. This spec would be wrong |

## Decision history

- ADR-0001: Modules are ordinary CUE unified into one Configuration. Rejected: a flake-style import map and export map. Rejected: a `modules.<name>.config` namespace.
- ADR-0002: The `package` clause is ignored. Rejected: requiring `package modot`.
- ADR-0003: An old leftover fails the run immediately. Rejected: a deprecation warning that still loads `workspaced.cue`, `workspaced.lock.json`, and `WORKSPACED_*`.
- ADR-0004: Linters and formatters are CUE `#CheckTool` values with one generic runner and a closed codec set. `codebase lint --review` emits GitHub Actions workflow commands for findings on the relevant diff. Rejected: one Go package per tool. Rejected: posting PR review comments from that command.
- ADR-0005: This repository exposes no supported Go API. Rejected: keeping `pkg/` on the import path.
- ADR-0006: ADR entries live in this file. `docs/specs/` is deleted after its live rules move here. Rejected: a `docs/adr/` tree. Rejected: keeping `docs/specs/` as a second law.
