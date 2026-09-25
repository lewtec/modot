---
name: cue-language
description: >
  CUE language semantics and typing for agents authoring or reviewing .cue
  files. Lattice model, types-as-values, unification, disjunction, defaults,
  structs/lists/bounds/definitions, field qualifiers, closedness. Not for
  modot apply/lock/roots/modules/tools alone (use modot skill). No
  package/import/module system (host projects supply that). Triggers: CUE,
  .cue, unify, disjunction, constraint, schema, #definition, defaults,
  cue eval, types are values, subsumption, bottom error. Secondary trigger:
  modot.cue only when the issue is language/unification, not product ops.
---

# CUE language (semantics & typing)

Condensed agent guide from the [CUE Language Specification](https://cuelang.org/docs/reference/spec/).
Out of scope here: `package` / `import` / modules / file layout — the host
environment (e.g. modot) owns how CUE is loaded and composed.

## With modot (sibling skill)

If the task is only modot product/ops (plan/apply, lock/tidy, config roots,
modules enablement, templates, tools, drivers) and not "why won't this CUE unify
/ type / default?", prefer `skills/modot/` and skip this skill — even when
the file is `modot.cue`.

If both apply (non-trivial CUE authoring for modot):

1. This skill — legal CUE (unify, defs, constraints, examples).
2. `skills/modot/references/cue.md` + `modules.md` — evaluated keys, merge
   edges, lock vs cue, inspect via modot config commands.
3. Always read the user's actual `modot.cue` (and modules); init/examples in
   either skill are not the local dialect.

Do not answer modot "wrong root / apply didn't run / lock pins" questions
from this skill alone.

## Start here

1. Types are values. `int`, `string`, `42`, and `{a: int}` live in one lattice.
2. Everything is a constraint. Data and schemas unify the same way.
3. Unification (`&`) = meet (greatest lower bound): tighten constraints.
4. Disjunction (`|`) = join (least upper bound): alternatives / sum types.
5. Top `_` accepts anything; bottom `_|_` is error / conflict / unsatisfiable.
6. `a` is an instance of `b` when `a` is more specific than (or equal to) `b` in
   the lattice. Concrete data sits at the bottom end; open types sit higher.

If two constraints disagree (e.g. `int & string`, or `1 & 2`), the result is
`_|_` (possibly with an error message in tooling).

## Load references by topic

| Question | Open |
|----------|------|
| Lattice, unify, disjunct, defaults, cycles | `references/semantics.md` |
| Basic kinds, literals, structs, lists, bounds, defs | `references/types-and-values.md` |
| `?`/`!`, closedness, patterns, comprehensions | `references/constraints.md` |
| Worked snippets | `references/examples.md` |

Paths relative to this skill directory (`skills/cue-language/`).

## Authoring rules of thumb

- Prefer unifying constraints over imperative checks: write what must hold.
- Keep merge-friendly shapes at composition boundaries: keyed structs over naked
  lists when layers/overlays unify. For modot specifically (module `config`
  merge physics, not a global CUE ban on lists):
  `skills/modot/references/cue.md` + `modules.md` — do not expand that
  contract here.
- Use definitions (`#Name`) for reusable schemas; they close by default.
- Use `*` defaults inside disjunctions only when one alternative should win when
  others are still incomplete.
- Optional vs required: `field?: T` (may be absent), `field!: T` (must exist),
  `field: T` (present with that constraint when the field appears).
- Failures are first-class (`_|_`), not exceptions — fix the conflicting conjuncts.

## Not this skill

- Modot product/ops (apply, lock, roots, modules, tools, drivers, templates):
  `skills/modot/` only — load this skill only if language/unification is
  also blocking.
- Modot host contracts for CUE (what gets decoded, merge edges, lock):
  `skills/modot/references/cue.md` (bridge; not a second language guide).
- CLI (`cue eval`, `cue export`, …): run the tool; this skill is language
  semantics. Modot may use different loaders/preludes than bare `cue eval`
  — product inspect: `modot home|codebase config …` (`--help`).
- Full grammar / lexer / implementor formalism: original spec only.
- Contributor schema/prelude/Go decode order: modot repo `AGENTS.md`.

## Source of truth

When in doubt, defer to https://cuelang.org/docs/reference/spec/ for exact
evaluation rules. This skill prioritizes agent-usable semantics over completeness.
