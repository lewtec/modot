package conda

import lewtool "github.com/lewtec/lewkit/x/tool"

func init() {
	lewtool.Register("conda", &Backend{})
}
