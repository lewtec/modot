package alacritty

import "github.com/lucasew/workspaced/pkg/driver/terminal"

func init() {
	terminal.RegisterExec("terminal_alacritty", "Alacritty", "alacritty", "-T", true, nil)
}
