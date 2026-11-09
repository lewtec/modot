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
