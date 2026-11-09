package view

import "github.com/a-h/templ"

// The file tree's own styling, as templ css classes served once by the
// middleware. The tree repeats these rules on a thousand rows, so they belong
// in one stylesheet rather than in a class attribute per cell.

// treeRule is one rule: a selector, by name, and the declarations it carries.
type treeRule struct {
	// name is the class name the middleware keys the rule on.
	name string
	// selector is the CSS selector, which templ cannot invent because it
	// keys off the tree's own data attributes and element classes.
	selector string
	// declarations are the CSS properties the rule sets.
	declarations string
}

// treeRules is every rule the file tree needs, in the order they are served: a
// later rule overrides an earlier one that matches the same node.
func treeRules() []treeRule {
	return []treeRule{
		{
			name:     "tree-row",
			selector: `[data-file-tree] .menu li > a, [data-file-tree] .menu li > button`,
			declarations: `
			display: flex;
			align-items: center;
			gap: 0.25rem;
			width: 100%;
			min-height: 1.5rem;
			padding: 0.125rem 0.5rem;
			border-radius: 0.25rem;
			font-size: 0.75rem;
			line-height: 1rem;
		`,
		},
		{
			name:     "tree-row-focus",
			selector: `[data-file-tree] .menu li > a:focus-visible, [data-file-tree] .menu li > button:focus-visible`,
			declarations: `
			outline: 2px solid var(--color-primary);
			outline-offset: -2px;
		`,
		},
		{
			name:     "tree-list",
			selector: `[data-file-tree] .menu ul`,
			declarations: `
			margin: 0;
			padding-left: 1rem;
		`,
		},
		{
			name:     "tree-group",
			selector: `[data-file-tree] .menu ul[role="group"]`,
			declarations: `
			border-left: 1px solid var(--color-base-300);
		`,
		},
		{
			name:     "tree-icon",
			selector: `[data-file-tree] svg`,
			declarations: `
			flex: none;
		`,
		},
		{
			name:     "tree-chevron",
			selector: `[data-file-tree] .tree-chevron`,
			declarations: `
			width: 0.75rem;
			height: 0.75rem;
			opacity: 0.4;
			transition: transform 120ms ease;
		`,
		},
		{
			name:     "tree-chevron-open",
			selector: `[data-file-tree] [aria-expanded="true"] .tree-chevron`,
			declarations: `
			transform: rotate(90deg);
		`,
		},
		{
			name:     "tree-folder",
			selector: `[data-file-tree] .tree-folder`,
			declarations: `
			width: 0.875rem;
			height: 0.875rem;
			opacity: 0.55;
		`,
		},
		{
			name:     "tree-leaf-gap",
			selector: `[data-file-tree] .tree-leaf-gap`,
			declarations: `
			width: 0.75rem;
			height: 0.75rem;
			opacity: 0.15;
		`,
		},
		{
			name:     "tree-leaf-icon",
			selector: `[data-file-tree] .tree-leaf-icon`,
			declarations: `
			width: 0.875rem;
			height: 0.875rem;
			opacity: 0.45;
		`,
		},
		{
			name:     "tree-meta",
			selector: `[data-file-tree] .tree-meta`,
			declarations: `
			display: flex;
			flex: none;
			align-items: center;
			gap: 0.5rem;
			margin-left: auto;
			padding-left: 0.5rem;
			font-family: var(--font-mono, ui-monospace, monospace);
			font-size: 0.6875rem;
			opacity: 0.55;
		`,
		},
		{
			name:     "tree-meta-source",
			selector: `[data-file-tree] .tree-meta-source`,
			declarations: `
			max-width: 10rem;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		`,
		},
		{
			name:     "tree-name",
			selector: `[data-file-tree] .tree-name`,
			declarations: `
			overflow: hidden;
			font-family: var(--font-mono, ui-monospace, monospace);
			text-overflow: ellipsis;
			white-space: nowrap;
		`,
		},
		{
			name:     "tree-count",
			selector: `[data-file-tree] .tree-count`,
			declarations: `
			margin-left: 0.25rem;
			font-size: 0.6875rem;
			font-variant-numeric: tabular-nums;
			opacity: 0.5;
		`,
		},
	}
}

// TreeClasses is the tree's rules as templ css classes for the middleware.
func TreeClasses() []templ.CSSClass {
	rules := treeRules()
	classes := make([]templ.CSSClass, 0, len(rules))
	for _, rule := range rules {
		body := rule.selector + "{" + rule.declarations + "}"
		classes = append(classes, templ.ComponentCSSClass{
			ID:    templ.CSSID(rule.name, body),
			Class: templ.SafeCSS(body),
		})
	}

	return classes
}
