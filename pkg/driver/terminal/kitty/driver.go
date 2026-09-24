package kitty

import "github.com/lucasew/workspaced/pkg/driver/terminal"

func init() {
	terminal.RegisterExec("terminal_kitty", "Kitty", "kitty", "--title", false, nil)
}
