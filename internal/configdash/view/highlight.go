package view

import (
	"bytes"
	"log/slog"
	"strings"

	"github.com/a-h/templ"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HighlightCSS is the stylesheet the highlighted code block needs. Chroma
// writes class names, not inline styles, so a large file does not repeat a
// style on every token; the palette lives here once per theme.
const HighlightCSS = `
.chroma { display: block; overflow-x: auto; }
.chroma .err { color: #a61717; background-color: #e3d2d2; }
.chroma .k, .chroma .kc, .chroma .kd, .chroma .kn, .chroma .kp, .chroma .kr, .chroma .kt, .chroma .kc, .chroma .kd, .chroma .kn, .chroma .kp, .chroma .kr, .chroma .kt, .chroma .nt { color: #a626a4; }
.chroma .s, .chroma .sa, .chroma .sb, .chroma .sc, .chroma .dl, .chroma .sd, .chroma .s2, .chroma .se, .chroma .sh, .chroma .si, .chroma .sx, .chroma .sr, .chroma .s1, .chroma .ss { color: #50a14f; }
.chroma .m, .chroma .mb, .chroma .mf, .chroma .mh, .chroma .mi, .chroma .il, .chroma .mo { color: #986801; }
.chroma .c, .chroma .ch, .chroma .cm, .chroma .c1, .chroma .cs, .chroma .cp, .chroma .cpf { color: #8e908c; font-style: italic; }
.chroma .nf, .chroma .fm, .chroma .nc { color: #4078f2; }
.chroma .nb, .chroma .bp, .chroma .nv, .chroma .vc, .chroma .vg, .chroma .vi, .chroma .vm { color: #e45649; }
.chroma .na, .chroma .nk, .chroma .no { color: #c18401; }
.chroma .gd { color: #a00000; }
.chroma .gi { color: #00a000; }
.chroma .gh, .chroma .gu { font-weight: bold; }
.chroma .gs { font-weight: bold; }
.chroma .ge { font-style: italic; }
[data-theme="dark"] .chroma .err { color: #f8f8f2; background-color: #7a1f1f; }
[data-theme="dark"] .chroma .k, [data-theme="dark"] .chroma .kc, [data-theme="dark"] .chroma .kd, [data-theme="dark"] .chroma .kn, [data-theme="dark"] .chroma .kp, [data-theme="dark"] .chroma .kr, [data-theme="dark"] .chroma .kt, [data-theme="dark"] .chroma .kc, [data-theme="dark"] .chroma .kd, [data-theme="dark"] .chroma .kn, [data-theme="dark"] .chroma .kp, [data-theme="dark"] .chroma .kr, [data-theme="dark"] .chroma .kt, [data-theme="dark"] .chroma .nt { color: #c678dd; }
[data-theme="dark"] .chroma .s, [data-theme="dark"] .chroma .sa, [data-theme="dark"] .chroma .sb, [data-theme="dark"] .chroma .sc, [data-theme="dark"] .chroma .dl, [data-theme="dark"] .chroma .sd, [data-theme="dark"] .chroma .s2, [data-theme="dark"] .chroma .se, [data-theme="dark"] .chroma .sh, [data-theme="dark"] .chroma .si, [data-theme="dark"] .chroma .sx, [data-theme="dark"] .chroma .sr, [data-theme="dark"] .chroma .s1, [data-theme="dark"] .chroma .ss { color: #98c379; }
[data-theme="dark"] .chroma .m, [data-theme="dark"] .chroma .mb, [data-theme="dark"] .chroma .mf, [data-theme="dark"] .chroma .mh, [data-theme="dark"] .chroma .mi, [data-theme="dark"] .chroma .il, [data-theme="dark"] .chroma .mo { color: #d19a66; }
[data-theme="dark"] .chroma .c, [data-theme="dark"] .chroma .ch, [data-theme="dark"] .chroma .cm, [data-theme="dark"] .chroma .c1, [data-theme="dark"] .chroma .cs, [data-theme="dark"] .chroma .cp, [data-theme="dark"] .chroma .cpf { color: #7f848e; }
[data-theme="dark"] .chroma .nf, [data-theme="dark"] .chroma .fm, [data-theme="dark"] .chroma .nc { color: #61afef; }
[data-theme="dark"] .chroma .nb, [data-theme="dark"] .chroma .bp, [data-theme="dark"] .chroma .nv, [data-theme="dark"] .chroma .vc, [data-theme="dark"] .chroma .vg, [data-theme="dark"] .chroma .vi, [data-theme="dark"] .chroma .vm { color: #e06c75; }
[data-theme="dark"] .chroma .na, [data-theme="dark"] .chroma .nk, [data-theme="dark"] .chroma .no { color: #e5c07b; }
`

// highlight renders text as Chroma-highlighted HTML. The lexer comes from the
// file's name, so a shell script or a JSON file reads right, and falls back to
// plain text: a wrong guess colors nothing rather than coloring it wrong.
func highlight(text, path string) templ.Component {
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		slog.Warn("dashboard: tokenise file", "path", path, "error", err)

		return plainCode(text)
	}
	var buffer bytes.Buffer
	// The palette is this package's CSS, so Chroma only writes class names
	// and never inline colors. A style is still needed: the formatter uses
	// it to decide which tokens are worth a span at all.
	formatter := html.New(html.WithClasses(true))
	formatErr := formatter.Format(&buffer, styles.Fallback, iterator)
	if formatErr != nil {
		slog.Warn("dashboard: format file", "path", path, "error", formatErr)

		return plainCode(text)
	}

	return rawHTML(buffer.String())
}

// plainCode is the fallback: escaped text in the block the highlighted form
// uses, so the page's styling still applies.
func plainCode(text string) templ.Component {
	var builder strings.Builder
	builder.WriteString(`<pre class="chroma"><code>`)
	builder.WriteString(templ.EscapeString(text))
	builder.WriteString(`</code></pre>`)

	return rawHTML(builder.String())
}

// rawHTML is markup this package generated itself. Highlighting output is the
// formatter's escaped HTML, never the file's bytes verbatim.
func rawHTML(markup string) templ.Component {
	return templ.Raw(markup)
}

// highlightStyle ships the palette with the page that uses it. It is built
// here rather than in the markup because templ does not interpolate inside a
// <style> element: the text is raw, so the style has to be a raw component.
func highlightStyle() templ.Component {
	return templ.Raw("<style>" + HighlightCSS + "</style>")
}
