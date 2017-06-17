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
	w.WriteString(string(p))
	return len(p), nil
}

// WriteString writes the provided string to the underlying interface.
func (w *PrettyWriter) WriteString(str string) {
	mu.Lock()
	stamp := color.BlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[%s]", w.name)
	msg := color.BlackString("- %s", str)
	fmt.Fprintf(w.file, "%s\r%s %s %s", cursor.ClearLine(), stamp, name, msg)
	mu.Unlock()
}

// WriteStringf writes the provided formatted string to the underlying
// interface.
func (w *PrettyWriter) WriteStringf(format string, args ...interface{}) {
	w.WriteString(fmt.Sprintf(format, args...))
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
	fmt.Fprintf(w.file, "%s\r%s", cursor.ClearLine(), string(p))
	mu.Unlock()
	return len(p), nil
}
