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

const (
	// When the child side of the pty is closed when it dies, the subsequent
	// read on ptmx is expected to fail.
	ptyErr = "read /dev/ptmx: input/output error"
)

var (
	mu                          = &sync.Mutex{}
	shouldOverwritePreviousLine = false
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
	writeLineToKeepWithPrefix(w.name, w.file, string(p))
	return len(p), nil
}

// WriteStringf writes the provided formatted string to the underlying
// interface.
func (w *PrettyWriter) WriteStringf(format string, args ...interface{}) {
	w.Write([]byte(fmt.Sprintf(format, args...)))
}

// Write implements the standard Write interface.
func (w *PrettyWriter) WriteAndFlagToReplace(p []byte) (int, error) {
	writeLineToBeReplacedWithPrefix(w.name, w.file, string(p))
	return len(p), nil
}

func writeString(file *os.File, str string) {
	if shouldOverwritePreviousLine {
		// In case the witch process is wrapped by some parent process with a log prefix (ex. docker compose)
		// we want to maintain any existing log prefix and only overwrite the rest of the line.
		// So we move up, clear and then write the new line.
		fmt.Fprintf(file, "%s\r%s", cursor.ClearLine, str)
	} else {
		fmt.Fprintf(file, "%s", str)
	}
}

func getLogPrefix(n string) string {
	stamp := color.HiBlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[%s]", n)
	wand := fmt.Sprintf("%s%s", color.GreenString("--"), color.MagentaString("⭑"))
	return fmt.Sprintf("%s %s %s", stamp, name, wand)
}

func writeLineToKeepWithPrefix(name string, file *os.File, format string, args ...interface{}) {
	msg := color.HiBlackString("%s", fmt.Sprintf(format, args...))
	output := fmt.Sprintf("%s %s", getLogPrefix(name), msg)

	mu.Lock()
	defer mu.Unlock()

	writeString(file, output)
	shouldOverwritePreviousLine = false
}

func writeLineToBeReplacedWithPrefix(name string, file *os.File, format string, args ...interface{}) {
	msg := color.HiBlackString("%s", fmt.Sprintf(format, args...))
	output := fmt.Sprintf("%s %s", getLogPrefix(name), msg)

	mu.Lock()
	defer mu.Unlock()

	writeString(file, output)
	shouldOverwritePreviousLine = true // set that we should replace this line on the next write
}

func writeLineToKeepWithoutPrefix(file *os.File, output string) {
	mu.Lock()
	defer mu.Unlock()

	writeString(file, output)
	shouldOverwritePreviousLine = false
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
	mu           *sync.Mutex
}

// NewCmd instantiates and returns a new cmd writer.
func NewCmd(name string, file *os.File) *CmdWriter {
	return &CmdWriter{
		name: name,
		file: file,
		kill: make(chan bool),
		mu:   &sync.Mutex{},
	}
}

// MaxTokenSize sets the max token size for the underlying scanner.
func (w *CmdWriter) MaxTokenSize(numBytes int) {
	w.maxTokenSize = numBytes
}

// Proxy will forward the output from the provided os.File through the writer.
func (w *CmdWriter) Proxy(f *os.File) {
	w.mu.Lock()
	// if we have an existing proxy, send EOF to kill it.
	if w.proxy != nil {
		// wait until its dead
		<-w.kill
	}
	// create new proxy
	w.proxy = f
	w.scanner = bufio.NewScanner(w.proxy)

	buf := make([]byte, w.maxTokenSize)
	w.scanner.Buffer(buf, w.maxTokenSize)

	w.mu.Unlock()
	go func() {
		for w.scanner.Scan() {
			line := w.scanner.Text()
			w.write([]byte(line + "\n"))
		}
		err := w.scanner.Err()
		if err != nil {
			if err.Error() != ptyErr {
				writeLineToKeepWithPrefix(w.name, w.file, "%s%s\n", color.HiRedString("proxy writer error: "), err.Error())
				os.Exit(3)
			}
		}
		w.kill <- true
	}()
}

// Write implements the standard Write interface.
func (w *CmdWriter) write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// append to buffer
	w.buffer += string(p)
	for {
		index := strings.IndexAny(w.buffer, "\n")
		if index == -1 {
			// no endline
			break
		}
		writeLineToKeepWithoutPrefix(w.file, fmt.Sprintf("%s", w.buffer[0:index+1]))
		w.buffer = w.buffer[index+1:]
	}
	return len(p), nil
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *CmdWriter) Flush() error {
	_, err := w.write([]byte(w.buffer))
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buffer) > 0 {
		writeLineToKeepWithoutPrefix(w.file, fmt.Sprintf("%s\n", w.buffer))
	}
	return nil
}
