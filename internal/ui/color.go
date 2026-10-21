package ui

import (
	"fmt"
	"os"
)

var (
	Green   = "\033[38;5;48m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Italic  = "\033[3m"
	Cyan    = "\033[36m"
	Yellow  = "\033[33m"
	Magenta = "\033[35m"
	Red     = "\033[31m"
	Gray    = "\033[90m"
	Reset   = "\033[0m"
)

var Disabled = false

func init() {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		DisableColors()
	}
}

func DisableColors() {
	Disabled = true
	Green = ""
	Bold = ""
	Dim = ""
	Italic = ""
	Cyan = ""
	Yellow = ""
	Magenta = ""
	Red = ""
	Gray = ""
	Reset = ""
}

func Colorize(color, text string) string {
	if Disabled {
		return text
	}
	return color + text + Reset
}

func Header(title string) string {
	return fmt.Sprintf("\n%s=== %s ===%s\n", Bold+Green, title, Reset)
}
