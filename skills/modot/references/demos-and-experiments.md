# Demos and experiments

## Why this file exists

Modot's CLI includes showcase / experiment entry points that are easy to
discover via tab-complete or casual `--help` browsing. They are not the primary
way to manage environments, tools, or drivers. This page keeps agents from
treating demos as the main API.

## Where demos live

### `modot experiments …`

Experimental / developer-oriented commands.

`modot experiments demo` (and subcommands) showcase output rendering and
lewkit `x/taskgroup` (progress, logs, optional TUI).

Examples of what that tree is for (names can grow; use `--help`):

| Invocation (pattern) | Intent |
|----------------------|--------|
| `experiments demo` / `… demo tasks` | Default taskgroup + UI showcase |
| `… demo plain` | Same scheduling without forcing the fancy renderer |
| `… demo nested` | Isolate error boundary + child tasks |
| `… demo loop` | Logs + progress interaction |
| `… demo map` | Parallel map-style tasks |

These demos document how modot itself renders work internally, not how to
configure modules or install tools.

There may be other experiments (e.g. cue-related); always inspect
`modot experiments --help` for the build you have.

### `modot utils demo …`

Under utils, smaller demos (progress, debug-style helpers, etc.). Same idea:
illustrative, not core workflow.

Other utils (`shell init`, `template materialize`, icons, palette, nix helpers,
history, …) are real utilities — not all of `utils` is "demo," but `utils demo`
specifically is.

## What is not a demo

Use these for real work:

| Area | Purpose |
|------|---------|
| `home` / `codebase` | Plan/apply/lint/format/config |
| `mod` | Lock refresh |
| `tool` | Tools and `tool with` |
| `driver` | Live OS actions |
| `open` | Open/launch helpers |
| `init` / `self-install` | Bootstrap |

If the task is "fix my dotfiles / run go 1.22 once / change volume," demos are
the wrong branch.

## When demos are appropriate

- Understanding modot's task/progress UI behavior.
- Developing or debugging modot's output layer (contributor context).
- Reproducing "what does the bubbletea path look like vs plain logs?"

They are optional for end users of modules/tools/drivers.

## Gotchas

- Agent maps `experiments demo` to "primary tutorial": it teaches task UI, not
  modules/templates/tools.
- Demo failures are synthetic or illustrative: simulated errors in demo code are
  not necessarily bugs in the user's config.
- `TERM=dumb` / CI / non-TTY: fancy demo UI may no-op to plain output; that
  doesn't mean modot is broken for normal commands.
- Utils vs experiments: both can contain demos; core product surface is still
  `home` / `codebase` / `mod` / `tool` / `driver`.
- Don't apply lessons from demo scheduling to module apply: different subsystems
  entirely (`plan-and-apply.md`).
