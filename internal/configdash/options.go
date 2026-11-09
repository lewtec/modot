package configdash

import (
	"sort"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"
)

// rootKeys names the top-level fields of an options expression. Fields nested
// in a struct are not root keys; a definition (#Thing) is listed too, so the
// dashboard can show it.
func rootKeys(expr string) []string {
	node, err := parser.ParseExpr("options.cue", expr)
	if err != nil {
		return nil
	}
	lit, ok := node.(*ast.StructLit)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(lit.Elts))
	for _, elt := range lit.Elts {
		field, ok := elt.(*ast.Field)
		if !ok {
			continue
		}
		label, _, err := ast.LabelName(field.Label)
		if err != nil {
			continue
		}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}
