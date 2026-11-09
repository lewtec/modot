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
