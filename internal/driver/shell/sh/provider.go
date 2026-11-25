package sh

import "github.com/lewtec/modot/internal/driver/shell"

func init() {
	shell.RegisterWhich("shell_sh", "POSIX sh", "sh")
}
