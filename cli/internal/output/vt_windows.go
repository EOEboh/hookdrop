//go:build windows

package output

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT switches the console to virtual-terminal (ANSI) mode.
// Windows Terminal has it on already; legacy conhost needs the flag.
func enableVT() bool {
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	return windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
