package writer

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/unchartedsoftware/witch/cursor"
)

var (
	mu = &sync.Mutex{}
)

// PrettyWriter represents a pretty formatteed writer
type PrettyWriter struct {
	file *os.File
	name string
}

// NewPretty instantiates and returns a new pretty writer.
func NewPretty(name string, file *os.File) *PrettyWriter {
	return &PrettyWriter{
		name: name,
		file: file,
	}
}

// Write implements the standard Write interface.
func (w *PrettyWriter) Write(p []byte) (int, error) {
	w.WriteStringf(string(p))
	return len(p), nil
}

// WriteStringf writes the provided formatted string to the underlying
// interface.
func (w *PrettyWriter) WriteStringf(format string, args ...interface{}) {
	stamp := color.BlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[%s]", w.name)
	wand := fmt.Sprintf("%s%s", color.GreenString("--"), color.MagentaString("⭑"))
	msg := color.BlackString("%s", fmt.Sprintf(format, args...))
	mu.Lock()
	fmt.Fprintf(w.file, "%s\r%s %s %s %s", cursor.ClearLine, stamp, name, wand, msg)
	mu.Unlock()
}

// CmdWriter represents a writer to log an output from the executed cmd.
type CmdWriter struct {
	file *os.File
}

// NewCmd instantiates and returns a new cmd writer.
func NewCmd(file *os.File) *CmdWriter {
	return &CmdWriter{
		file: file,
	}
}

// Write implements the standard Write interface.
func (w *CmdWriter) Write(p []byte) (int, error) {
	mu.Lock()
	fmt.Fprintf(w.file, "%s\r%s", cursor.ClearLine, string(p))
	mu.Unlock()
	return len(p), nil
}
