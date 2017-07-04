package writer

import (
	"fmt"
	"os"
	"strings"
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
	file   *os.File
	buffer string
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
	// append to buffer
	w.buffer += string(p)
	for {
		index := strings.IndexAny(w.buffer, "\n")
		if index == -1 {
			// no endline
			break
		}
		fmt.Fprintf(w.file, "%s\r%s", cursor.ClearLine, w.buffer[0:index+1])
		w.buffer = w.buffer[index+1:]
	}
	mu.Unlock()
	return len(p), nil
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *CmdWriter) Flush() error {
	_, err := w.Write([]byte(w.buffer))
	if err != nil {
		return err
	}
	mu.Lock()
	if len(w.buffer) > 0 {
		fmt.Fprintf(w.file, "%s\r%s\n", cursor.ClearLine, w.buffer)
	}
	mu.Unlock()
	return nil
}
