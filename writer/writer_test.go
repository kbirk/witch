package writer

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func TestProxyReplacementDoesNotBlockProxyFlush(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "witch-output-*")
	if err != nil {
		t.Fatalf("failed to create output file: %v", err)
	}
	defer output.Close()

	cmdWriter := NewCmd("witch", output)
	cmdWriter.MaxTokenSize(1024)

	oldReader, oldWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create old pipe: %v", err)
	}
	defer oldReader.Close()
	cmdWriter.Proxy(oldReader)

	newReader, newWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create new pipe: %v", err)
	}
	defer newReader.Close()
	defer newWriter.Close()

	replaced := make(chan struct{})
	go func() {
		cmdWriter.Proxy(newReader)
		close(replaced)
	}()

	deadline := time.After(50 * time.Millisecond)
waiting:
	for {
		select {
		case <-deadline:
			break waiting
		default:
			if cmdWriter.mu.TryLock() {
				cmdWriter.mu.Unlock()
				runtime.Gosched()
				continue
			}
			break waiting
		}
	}

	if _, err := oldWriter.WriteString("old output\n"); err != nil {
		t.Fatalf("failed to write old output: %v", err)
	}
	if err := oldWriter.Close(); err != nil {
		t.Fatalf("failed to close old writer: %v", err)
	}

	select {
	case <-replaced:
	case <-time.After(time.Second):
		t.Fatal("proxy replacement blocked while the previous proxy flushed output")
	}
}

func TestFlushWritesPartialLineOnce(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "witch-output-*")
	if err != nil {
		t.Fatalf("failed to create output file: %v", err)
	}
	defer output.Close()

	cmdWriter := NewCmd("witch", output)
	if _, err := cmdWriter.write([]byte("partial")); err != nil {
		t.Fatalf("failed to buffer partial output: %v", err)
	}
	if err := cmdWriter.Flush(); err != nil {
		t.Fatalf("failed to flush partial output: %v", err)
	}
	if err := cmdWriter.Flush(); err != nil {
		t.Fatalf("failed to flush empty buffer: %v", err)
	}

	contents, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(contents) != "partial\n" {
		t.Fatalf("flush output = %q, want %q", contents, "partial\n")
	}
	if cmdWriter.buffer != "" {
		t.Fatalf("buffer = %q, want empty", cmdWriter.buffer)
	}
}
