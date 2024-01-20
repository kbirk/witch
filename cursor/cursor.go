package cursor

const (
	// Show returns ANSI escape sequence to show the cursor
	Show = "\x1b[?25h"
	// Hide hides the terminal cursor.
	Hide = "\x1b[?25l"
	// ClearLine clears the current line.
	ClearLine = "\x1b[2K"
	// ClearToRight clears the screen from the cursor to the end of the line.
	ClearToRight = "\x1b[0J"
	// MoveCursorLeft moves the cursor left one character.
	MoveCursorLeft = "\x1b[D"
	// MoveCursorUp moves the cursor up one line.
	MoveCursorUp = "\x1b[1A"
	// MoveCursorDown moves the cursor down one line.
	MoveCursorDown = "\x1b[1B"
)
