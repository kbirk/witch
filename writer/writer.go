package writer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/kbirk/witch/cursor"
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

func writeStringf(n string, file *os.File, format string, args ...interface{}) {
	stamp := color.HiBlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[%s]", n)
	wand := fmt.Sprintf("%s%s", color.GreenString("--"), color.MagentaString("⭑"))
	msg := color.HiBlackString("%s", fmt.Sprintf(format, args...))
	mu.Lock()
	fmt.Fprintf(file, "%s\r%s %s %s %s", cursor.ClearLine, stamp, name, wand, msg)
	mu.Unlock()
}

// WriteStringf writes the provided formatted string to the underlying
// interface.
func (w *PrettyWriter) WriteStringf(format string, args ...interface{}) {
	writeStringf(w.name, w.file, format, args...)
}

// CmdWriter represents a writer to log an output from the executed cmd.
type CmdWriter struct {
	name         string
	file         *os.File
	proxy        *os.File
	scanner      *bufio.Scanner
	maxTokenSize int
	buffer       string
	kill         chan bool
}

// NewCmd instantiates and returns a new cmd writer.
func NewCmd(name string, file *os.File) *CmdWriter {
	return &CmdWriter{
		name: name,
		file: file,
		kill: make(chan bool),
	}
}

// MaxTokenSize sets the max token size for the underlying scanner.
func (w *CmdWriter) MaxTokenSize(numBytes int) {
	w.maxTokenSize = numBytes
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
