# Workspaced code map

Read this before editing. Layout rule is in AGENTS.md (`pkg/` vs `internal/` vs `cmd/`).

## Architecture

CUE config (`workspaced.cue`) drives everything.

- Drivers (`pkg/driver`): OS features (audio, clipboard, WM, …). One impl per interface, chosen by weights and compatibility checks.
- Modules + source pipeline (`internal/module`, `internal/source`): config and templates become real files, streamed in memory.
- Tools: `github.com/lewtec/lewkit/x/tool` (github, mise, registry) plus lewkit fetchurl/httpclient drivers. Workspaced `internal/tool` owns the store path, shims, and lock rows.
- Checks (`internal/checks`): CUE-defined linters/formatters (`lint`/`formatter` tools + codecs); `lint --review` → GHA workflow annotations.
- CLI packages under `cmd/workspaced/` are small x/cmd structs.

## Critical locations

- `pkg/driver/driver.go` and `pkg/driver/prelude`: driver system
- Task pools / progress: `github.com/lewtec/lewkit/x/taskgroup` and `x/taskgroup/progress` (AGENTS.md map/reduce rule)
- `pkg/palette/`, `pkg/logging/`, `pkg/api/`, `pkg/filespine/`: rest of `pkg/` (profile visibility and template paths)
- `internal/tool`: store path, shims, lazy lock pins (`Pin` → Renovate fields). Backends register from `x/tool/prelude`.
- Registry install check: `mise run test:registry-install` (sets `WORKSPACED_TEST_TOOL_INSTALL=1`) installs each lewkit catalog tool. Not part of `mise release`. Multi-target CI: `.github/workflows/registry-install.yml` (linux amd64 + arm64). Per-tool failures append to `GITHUB_STEP_SUMMARY` from `internal/tool/registry_install_test.go`.
- `internal/configcue/`, `internal/modfile/`, `internal/source/`: config, state, rendering; dest bytes are `github.com/lewtec/lewkit/x/fs/compose` (`docs/specs/file-spine.md`)
- `internal/db/`: sqlite store. Queries in `sqlite/*.sql`, migrations in `sqlite/migrations/`. `go generate ./internal/db` runs lewkit generate db.
- `internal/apply/`, `internal/deployer/`: apply flow
- `cmd/workspaced/root.go`: `pkg/driver/prelude`; tool/check preludes load from the cmds that need them

## Registration (`init()` based)

- Drivers: `driver.Register[T](impl)`
- Tool backends: `github.com/lewtec/lewkit/x/tool`.Register (via `x/tool/prelude`)
- Curated tools: `x/tool/registry`.RegisterTool
- CLI subcommands: `type Command struct` fields (wired by generated `children`)

Never import driver prelude except from `cmd/workspaced/root.go`. Tool/check preludes: from the cmd that needs them, not from `pkg/`. CLI groups export `Command` and embed generated `children`.

## Common tasks

- New driver capability: `pkg/driver/newthing/` (interface + facade + impl), then add to prelude.
- Curated tool: `github.com/lewtec/lewkit/x/tool/registry/applications/`
- Change how a tool locks: that tool's `Pin` method. Workspaced copies `Pin` onto the lock row.
- CUE schema changes: `internal/configcue/schema.cue`

Rules live in AGENTS.md. An older long-table version of this file may still be in git history if you need it.

## Terminology

See README.md (one sentence per term). Prefer factory (drivers), backend (tools), check (linters/formatters). Leftover "provider" wording is mostly module sources (`SourceProvider`) and transitional tool backend types.

## Doc style

For complex subsystems, mirror `skills/workspaced/references/templates.md`: decision tree, concrete examples, short reference tables. Usage skill material goes under `skills/workspaced/`.

`AGENTS.md` has the "when you do X, touch these files in this order" lists.

## Gotchas (also in AGENTS.md)

- No lists in module configs.
- Use `pkg/driver/exec` outside driver implementations.
- Import driver prelude only from `cmd/workspaced/root.go`.
- Only leaf taskgroup tasks take IO/CPU/Internet (`x/taskgroup` package doc).

Rest is in AGENTS.md and `skills/workspaced/`.

Update this file when structure shifts in a big way.
