package view

import (
	"strings"

	"github.com/a-h/templ"
)

// The syntax palette, one templ css class per token group.
//
// Chroma emits its own class names ("k", "s2", "nf"), which templ cannot
// generate, so the `css` block syntax does not fit: it always declares
// properties under a name it invents. A ComponentCSSClass is the type those
// blocks produce, and the middleware writes its Class verbatim into the
// stylesheet it serves, so a rule can select a Chroma class directly and the
// token markup needs nothing added to it.
//
// The classes are handed to templ's CSS middleware, which serves them once at
// /styles/templ.css and stops templ inlining them on every page.

// chromaGroup is one set of Chroma token classes painted the same way.
type chromaGroup struct {
	// name is the class name the middleware keys the rule on.
	name string
	// tokens are Chroma's own class names for the tokens in the group.
	tokens []string
	// declarations are the CSS properties, light theme.
	declarations string
	// darkDeclarations are the same properties for the dark theme. Empty
	// reuses the light ones.
	darkDeclarations string
}

// chromaPalette is every token class the page colors. A token not listed keeps
// the page's own text color.
func chromaPalette() []chromaGroup {
	return []chromaGroup{
		{
			name:             "chroma-k",
			tokens:           []string{"k", "kc", "kd", "kn", "kp", "kr", "kt", "nt"},
			declarations:     "color:#a626a4;",
			darkDeclarations: "color:#c678dd;",
		},
		{
			name:             "chroma-s",
			tokens:           []string{"s", "sa", "sb", "sc", "dl", "sd", "s2", "se", "sh", "si", "sx", "sr", "s1", "ss"},
			declarations:     "color:#50a14f;",
			darkDeclarations: "color:#98c379;",
		},
		{
			name:             "chroma-m",
			tokens:           []string{"m", "mb", "mf", "mh", "mi", "il", "mo"},
			declarations:     "color:#986801;",
			darkDeclarations: "color:#d19a66;",
		},
		{
			name:             "chroma-c",
			tokens:           []string{"c", "ch", "cm", "c1", "cs", "cp", "cpf"},
			declarations:     "color:#8e908c;font-style:italic;",
			darkDeclarations: "color:#7f848e;font-style:italic;",
		},
		{
			name:             "chroma-nf",
			tokens:           []string{"nf", "fm", "nc"},
			declarations:     "color:#4078f2;",
			darkDeclarations: "color:#61afef;",
		},
		{
			name:             "chroma-nv",
			tokens:           []string{"nb", "bp", "nv", "vc", "vg", "vi", "vm"},
			declarations:     "color:#e45649;",
			darkDeclarations: "color:#e06c75;",
		},
		{
			name:             "chroma-na",
			tokens:           []string{"na", "nk", "no"},
			declarations:     "color:#c18401;",
			darkDeclarations: "color:#e5c07b;",
		},
		{
			name:             "chroma-o",
			tokens:           []string{"o", "ow", "p"},
			declarations:     "color:#383a42;",
			darkDeclarations: "color:#abb2bf;",
		},
		{
			name:             "chroma-err",
			tokens:           []string{"err"},
			declarations:     "color:#a61717;background-color:#e3d2d2;",
			darkDeclarations: "color:#f8f8f2;background-color:#7a1f1f;",
		},
		{
			// A diff line is happy in either theme, so it has no dark twin.
			name:             "chroma-gi",
			tokens:           []string{"gi"},
			declarations:     "color:#00a000;",
			darkDeclarations: "",
		},
		{
			name:             "chroma-gd",
			tokens:           []string{"gd"},
			declarations:     "color:#a00000;",
			darkDeclarations: "",
		},
		{
			// Headings and subheadings, which markup and reST produce.
			name:             "chroma-gh",
			tokens:           []string{"gh", "gu"},
			declarations:     "font-weight:bold;",
			darkDeclarations: "",
		},
		{
			// Strong text.
			name:             "chroma-gs",
			tokens:           []string{"gs"},
			declarations:     "font-weight:bold;",
			darkDeclarations: "",
		},
		{
			name:             "chroma-ge",
			tokens:           []string{"ge"},
			declarations:     "font-style:italic;",
			darkDeclarations: "",
		},
	}
}

// chromaSelector is the selector for one group under a theme. An empty theme
// matches either, which is how a rule without a dark twin is written.
func chromaSelector(theme string, tokens []string) string {
	selectors := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if theme == "" {
			selectors = append(selectors, ".chroma ."+token)

			continue
		}
		selectors = append(selectors, `[data-theme="`+theme+`"] .chroma .`+token)
	}

	return strings.Join(selectors, ", ")
}

// chromaClass builds the class for one selector and its declarations. The body
// is what the middleware writes, so it is a whole rule, not properties.
func chromaClass(name, selector, declarations string) templ.ComponentCSSClass {
	body := selector + "{" + declarations + "}"

	return templ.ComponentCSSClass{
		ID:    templ.CSSID(name, body),
		Class: templ.SafeCSS(body),
	}
}

// chromaBlock is the code block itself: a block that scrolls, never a line that
// wraps, so a long line stays one line.
func chromaBlock() templ.ComponentCSSClass {
	return chromaClass("chroma-block", ".chroma", "display:block;overflow-x:auto;")
}

// ChromaClasses is every syntax rule, light rules first and dark rules after,
// so a dark rule wins where both select the same token.
func ChromaClasses() []templ.CSSClass {
	classes := []templ.CSSClass{chromaBlock()}
	for _, group := range chromaPalette() {
		classes = append(classes, chromaClass(group.name, chromaSelector("", group.tokens), group.declarations))
	}
	for _, group := range chromaPalette() {
		if group.darkDeclarations == "" {
			continue
		}
		dark := chromaClass(group.name+"-dark", chromaSelector("dark", group.tokens), group.darkDeclarations)
		classes = append(classes, dark)
	}

	return classes
}

// ChromaTokens lists every Chroma token class the palette covers, so a test can
// check the stylesheet against what the formatter actually emits.
func ChromaTokens() []string {
	var tokens []string
	for _, group := range chromaPalette() {
		tokens = append(tokens, group.tokens...)
	}

	return tokens
}
