package cursor

import (
	"fmt"
)

var (
	esc = "\x1b"
)

// Show returns ANSI escape sequence to show the cursor
func Show() string {
	return escape("[?25h")
}

// Hide hides the terminal cursor.
func Hide() string {
	return escape("[?25l")
}

// ClearLine clears the terminal line.
func ClearLine() string {
	return escape("[2K")
}

func escape(format string, args ...interface{}) string {
	return fmt.Sprintf("%s%s", esc, fmt.Sprintf(format, args...))
}
