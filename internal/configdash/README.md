# modot config dashboard

A web view of how modot's configuration merges: the layers it discovered, the
modules that inject their `options` schema into the shared tree, and the merged
`Configuration` with the file behind each key.

## Run it

```sh
modot dashboard            # codebase layers, web view when a desktop session exists
modot dashboard --no-open  # HTTP on a random loopback port, URL on stdout
modot dashboard --addr 127.0.0.1:8799
modot dashboard --scope home
```

`--scope home|codebase` picks the layer set and defaults to `codebase`. `--cwd`
sets where discovery starts.

## Pages

- `/` — stat cards (layers, enabled modules, merged keys, keys a module supplied),
  the discovery order, the top-level keys, and the module list.
- `/layers` — every `modot.cue` found, in merge order.
- `/modules?name=NAME` — a module's declared `options` schema next to what it
  resolved to.
- `/merged?path=KEY` — the merged tree. Each leaf carries a badge naming the
  module, or the file, that supplied it.
- `/merged.json` — the merged `Configuration` as JSON.
- `/files` — the files `modot <scope> apply` would place, with the module,
  place entry, or config-tree path behind each one.

## Materialized files

`/files` answers a different question from the merged tree: not what the config
*means*, but what lands on disk. `LoadFiles` evaluates the same config, then
runs the apply pipeline (source scanners, template rendering, file profiles,
place mounts) and lists the composed tree without writing anything.

The tree is built for the scope's own apply root — `~` for home, the workspace
for codebase, `/` for system — and only the file profiles that mode makes
visible. It is the tree, not a diff: whether a file changes needs the state
file, which a preview does not read.

`file.<profile>.<dest>` place entries keep their key as the file's source, so
the page can name the entry a tree of mounted files came from.

The page is separate because building the tree renders every template and can
fetch module sources; the other pages read only the merge.

## How a key gets its source

`internal/configdash` evaluates the config with the normal loader. Two rules
attribute the result:

1. **Structure** — `options.<name>` belongs to module `<name>`, and a top-level
   key belongs to the module whose `options` schema declares it.
2. **File** — `configcue.EvaluationResult.Provenance` walks the evaluated value
   and records `cue.Value.Source()`: the file of the conjunct that last set the
   value. CUE unifies conjuncts rather than overriding them, so two layers
   disagreeing on a value is a merge error, not a silent winner.

A load failure still renders: `Load` returns the layers and the modules alongside
the error, and the pages show them under the error alert.

## Development

```sh
go tool templ generate ./internal/configdash/view   # after editing *.templ
go test ./internal/configdash/ ./internal/configcue/
```

The UI is templ plus the daisyUI and Tailwind CSS assets the lewkit asset
package serves under `/__lewkit__/`. The theme is a `modot_theme` cookie read on
each request, so the choice survives navigation without server state: `system`
(the default) follows `prefers-color-scheme`, `light` and `dark` pin a palette.
Home, codebase, and system are tabs in the header, and each scope is its own
route (`/home/layers`, `/system/modules?name=icons`), so the URL always says what
is on screen.
