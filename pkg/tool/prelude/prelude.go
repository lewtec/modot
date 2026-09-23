// Package prelude registers lewkit tool backends so
// github.com/lewtec/lewkit/x/tool can resolve specs such as registry:mise
// without the CLI process.
//
// Blank-import this package from external programs. The workspaced CLI loads
// the same lewkit prelude from cmd/workspaced/root.go via internal/tool/prelude.
package prelude

import (
	_ "github.com/lewtec/lewkit/x/tool/prelude"
)
