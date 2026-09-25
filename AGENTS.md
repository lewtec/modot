# Modot development

Read in this order:
1. [SPEC.md](SPEC.md) — the constitution.
2. [README.md](README.md) — terms in one line each, plus a thin architecture sketch.
3. [CODEMAP.md](CODEMAP.md) — where things live and how to find them.
4. This file — rules that break things if you ignore them.
5. [skills/modot/references/templates.md](skills/modot/references/templates.md) — read before editing any template (kinds, functions, pitfalls).

## Package layout

- `internal/` — the program. CLI, apply, tools, modules, checks, config, drivers, helpers.
- `cmd/modot/` — x/cmd entrypoints only.
- New code goes in `internal/<domain>/`. There is no supported import path outside this repository.
- Name packages by job. No `utils` / `common` buckets.

## Hard rules

Break these and behavior will be wrong.

- Config is CUE-first only. Do not fall back to `GlobalConfig.Merge()` patterns.
- Adding a config field, in order:
  1. Update schema + defaults in `internal/configcue/schema.cue` (and preambles if needed).
  2. Add the key in the user's `modot.cue`.
  3. Decode with `configcue.Config.Decode()` or `ModuleConfig()`.
  4. Consume via `{{ .Field }}` in templates.
- No lists in module configs. Shapes must deep-merge cleanly.
- Driver preloads: import `_ "github.com/lewtec/modot/internal/driver/prelude"` only from `cmd/modot/root.go`. Never duplicate that import in subcommands.
- Tool preloads (`internal/tool/prelude`): import from the cmd that needs registration.
- Linters/formatters: CUE `lint` / `formatter` (SPEC.md ADR-0004). No per-tool Go packages.
- Process execution outside driver implementations goes through `internal/driver/exec`.
- Network: `fetchurl` driver when you have a hash, otherwise `httpclient`. Do not use `http.DefaultClient` directly.
- Tool backends: scope on backends (github, mise, catalog), not on individual tools. Prefer the word "backend".
- Module processing streams in memory. No intermediate files on disk.
- Markdown goes stale. Point at code instead of copying facts into docs.

## Driver system

Drivers implement `DriverFactory[T]`:
- `ID()`, `Name()`, `CheckCompatibility()`, `New()`

`driver.Get[T](ctx)` picks an impl using weights from `modot.cue`.

Register an impl by importing its package from the central prelude.

## When adding things

- New driver category: `internal/driver/<cat>/driver.go` (interface) + `facade.go`, one impl dir, import in `internal/driver/prelude/prelude.go`, CLI under `cmd/modot/driver/<cat>/`.
- Tool backends, the curated catalog, and tool download drivers (fetchurl, httpclient) live in `github.com/lewtec/lewkit/x/tool` and `x/driver`. Modot keeps the store path, shims, lazy locks, and CLI. Add a backend in lewkit, then blank-import it from `internal/tool/prelude` (which loads `x/tool/prelude`).
- New module source: implement under `internal/modfile/sourceprovider/`.
- Anything else: `internal/<domain>/` unless another module should import it.

## Module lock model

- `modot.cue` holds declarative inputs.
- `modot.lock.json` holds resolved pins (source URLs, hashes, tool versions).
  - Aside from `ref` and `kind`, fields are Renovate hints.
    - `kind = tool`
      - `ref` => tool ref (e.g. modot tool)
      - `source` => key into `inputs`

## Anti-patterns

- Duplicating prelude imports.
- A supported import path outside this repository.
- Raw `os/exec` or `http.DefaultClient` in feature code.
- Treating tools as first-class backends instead of going through registries/backends.
- Lists in module config.
- `utils` / `common` package names.
- `Each` + `mu.Lock` + `append` to build a result list — use `Map` + a pure reduce.
- Hand-rolled `WaitGroup`/`chan` fan-out when a `taskgroup.Session` is already on `ctx`.

Locate-X recipes live in CODEMAP.md.

## Patterns

- The context argument is always named `ctx`.
- `logger = logging.GetLogger(ctx)`. Do not import `log/slog` directly.
- An inner scope must not reuse an outer scope's `logger` or `ctx`.
- Prefer channels over locked shared state when that keeps the code simpler (it often does).
- `context.Background` and friends need a real reason. "No context in scope" is not one.
- Test root ctx: `logging.NewWriterContext(t.Output())` (or `b.Output()` / `io.Discard`). Not `NewRootContext(nil)` — that is `slog.Default()` on stderr.
- Pipeable data goes to stdout; everything else to stderr. Stdout is for one-line-per-record output (line-oriented text, JSONL) that another program can consume without multi-line parsing.

### Taskgroup map/reduce

Parallel work over a list goes through `github.com/lewtec/lewkit/x/taskgroup` (package doc: one owner per bar; `Isolate` / `GoIsolated` / `Map` / `Each`).

- `Map[T,U].Run`: fan-out that returns `[]U` in input order. Reduce with a pure merge (`errors.Join`, `BundleRuns`, ordered lockfile writes, state patches).
- `Each[T].Run`: fan-out when only success/failure matters (no `struct{}` results).
- If you reach for `Each` + mutex + `append`, switch to `Map` + pure reduce. Shared mutable state touched from parallel FS/network work should return a patch from the map step; apply patches serially in the reduce step.
- Do not wrap `Map`/`Each` in an extra `Control`+`Unit` shell when they already own the aggregate bar.
- Only leaf tasks take IO/CPU/Internet. Orchestrators (`Map`/`Each`/`GoIsolated`/`Go` whose Fn schedules more taskgroup or httpclient/rsync work) are Control. Same-kind nest deadlocks when the pool fills. Details: `x/taskgroup` package doc.
