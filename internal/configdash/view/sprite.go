package view

import (
	"regexp"
	"strings"

	"github.com/a-h/templ"
	"github.com/lewtec/lewkit/x/http/asset/lucide"
)

// iconOpen is the opening tag of an icon's SVG element. The sprite keeps only
// what a <symbol> needs: the viewBox positions the drawing, and the stroke
// attributes come from the element that references it.
var iconOpen = regexp.MustCompile(`(?s)^<svg[^>]*>`)

// iconClose is the closing tag.
const iconClose = "</svg>"

// spriteIcon draws an icon from the page's sprite. The sprite is built from
// the same catalog lewkit ships, so an unknown name still draws nothing; the
// difference is one symbol per page instead of one inline SVG per use, which
// matters when a tree lists a thousand files.
func spriteIcon(name lucide.Name, class string) templ.Component {
	return templ.Raw(`<svg class="` + templ.EscapeString(class) + `" aria-hidden="true"><use href="#icon-` + name.String() + `"></use></svg>`)
}

// iconSprite defines every icon a page uses, once, as a <symbol>. An icon the
// catalog does not know is skipped; there is nothing to draw.
func iconSprite(names []lucide.Name) templ.Component {
	var builder strings.Builder
	builder.WriteString(`<svg class="hidden" aria-hidden="true" focusable="false"><defs>`)
	seen := map[lucide.Name]bool{}
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		body, ok := lucide.SVG(name)
		if !ok {
			continue
		}
		inner := iconOpen.ReplaceAllString(body, "")
		inner = strings.TrimSuffix(strings.TrimSpace(inner), iconClose)
		builder.WriteString(`<symbol id="icon-` + name.String() + `" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`)
		builder.WriteString(inner)
		builder.WriteString(`</symbol>`)
	}
	builder.WriteString(`</defs></svg>`)

	return templ.Raw(builder.String())
}

// treeIcons lists the icons the file tree uses, so the sprite carries exactly
// those and the markup refers to them by name.
func treeIcons() []lucide.Name {
	return []lucide.Name{
		lucide.NameChevronRight,
		lucide.NameFolder,
		lucide.NameFile,
		lucide.NameFileCode,
		lucide.NameFileSymlink,
	}
}

// TreeCSS styles the file tree's own parts. The tree repeats these on every
// row, so they live in a stylesheet rather than in a class attribute per
// element: a thousand rows of repeated classes is most of the page.
const TreeCSS = `
[data-file-tree] .menu li > a,
[data-file-tree] .menu li > button {
	display: flex;
	align-items: center;
	gap: 0.25rem;
	width: 100%;
	min-height: 1.5rem;
	padding: 0.125rem 0.5rem;
	border-radius: 0.25rem;
	font-size: 0.75rem;
	line-height: 1rem;
}
[data-file-tree] .menu li > a:focus-visible,
[data-file-tree] .menu li > button:focus-visible {
	outline: 2px solid var(--color-primary);
	outline-offset: -2px;
}
[data-file-tree] .menu ul {
	margin: 0;
	padding-left: 1rem;
}
[data-file-tree] .menu ul[role="group"] {
	border-left: 1px solid var(--color-base-300);
}
[data-file-tree] svg {
	flex: none;
}
[data-file-tree] .tree-chevron {
	width: 0.75rem;
	height: 0.75rem;
	opacity: 0.4;
	transition: transform 120ms ease;
}
[data-file-tree] [aria-expanded="true"] .tree-chevron {
	transform: rotate(90deg);
}
[data-file-tree] .tree-folder {
	width: 0.875rem;
	height: 0.875rem;
	opacity: 0.55;
}
[data-file-tree] .tree-leaf-gap {
	width: 0.75rem;
	height: 0.75rem;
	opacity: 0.15;
}
[data-file-tree] .tree-leaf-icon {
	width: 0.875rem;
	height: 0.875rem;
	opacity: 0.45;
}
[data-file-tree] .tree-meta {
	display: flex;
	flex: none;
	align-items: center;
	gap: 0.5rem;
	margin-left: auto;
	padding-left: 0.5rem;
	font-family: var(--font-mono, ui-monospace, monospace);
	font-size: 0.6875rem;
	opacity: 0.55;
}
[data-file-tree] .tree-meta-source {
	max-width: 10rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
[data-file-tree] .tree-name {
	overflow: hidden;
	font-family: var(--font-mono, ui-monospace, monospace);
	text-overflow: ellipsis;
	white-space: nowrap;
}
[data-file-tree] .tree-count {
	margin-left: 0.25rem;
	font-size: 0.6875rem;
	font-variant-numeric: tabular-nums;
	opacity: 0.5;
}
`

// treeStyle ships the tree's stylesheet with the page that uses it.
func treeStyle() templ.Component {
	return templ.Raw("<style>" + TreeCSS + "</style>")
}
