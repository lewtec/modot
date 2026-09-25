package bash

import "github.com/lewtec/modot/internal/driver/shell"

func init() {
	shell.RegisterWhich("shell_bash", "Bash", "bash")
}
