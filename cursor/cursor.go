package cursor

import (
	"fmt"
)

var (
	esc = "\x1b"
)

// ShowCursor shows the terminal cursor.
func ShowCursor() string {
	return escape("[?25h")
}

// HideCursor hides the terminal cursor.
func HideCursor() string {
	return escape("[?25l")
}

// ClearLine clears the terminal line.
func ClearLine() string {
	return escape("[2K")
}

func escape(format string, args ...interface{}) string {
	return fmt.Sprintf("%s%s", esc, fmt.Sprintf(format, args...))
}
