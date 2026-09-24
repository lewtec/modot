package xclip

import "github.com/lucasew/workspaced/pkg/driver/clipboard"

func init() {
	clipboard.RegisterCmd(
		"clipboard_xclip",
		"X11 (xclip)",
		"xclip",
		nil,
		[]string{"-selection", "clipboard", "-t", "image/png"},
		[]string{"-selection", "clipboard"},
	)
}
