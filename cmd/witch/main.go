package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/fatih/color"

	"github.com/kbirk/witch/graceful"
	"github.com/kbirk/witch/spinner"
	"github.com/kbirk/witch/watcher"
	"github.com/kbirk/witch/writer"
)

const (
	name    = "witch"
	version = "0.2.13"
)

var (
	watch         []string
	ignore        []string
	cmd           string
	watchInterval int
	noSpinner     bool
	stopOnNonZero bool
	maxTokenSize  int
	tickInterval  = 100
	prev          *exec.Cmd
	ready         = make(chan bool, 1)
	mu            = &sync.Mutex{}
	prettyWriter  = writer.NewPretty(name, os.Stdout)
	cmdWriter     = writer.NewCmd(name, os.Stdout)
	spin          = spinner.New(prettyWriter)
)

func createLogo() string {
	return color.GreenString("\n        \\    / ") +
		color.MagentaString("★") +
		color.GreenString(" _|_  _ |_\n         ") +
		color.GreenString("\\/\\/  |  |_ |_ | |\n\n        ") +
		color.HiBlackString("version %s\n\n", version)
}

func fileChangeString(path string, event watcher.EventType) string {
	switch event {
	case watcher.Added:
		return fmt.Sprintf("%s %s",
			color.HiBlackString(path),
			color.GreenString("added"))
	case watcher.Removed:
		return fmt.Sprintf("%s %s",
			color.HiBlackString(path),
			color.RedString("removed"))
	}
	return fmt.Sprintf("%s %s",
		color.HiBlackString(path),
		color.BlueString("changed"))
}

func fileCountString(count uint64) string {
	switch count {
	case 0:
		return color.HiBlackString("no files found")
	case 1:
		return fmt.Sprintf("%s %s %s",
			color.HiBlackString("watching"),
			color.BlueString("%d", count),
			color.HiBlackString("file"))
	}
	return fmt.Sprintf("%s %s %s",
		color.HiBlackString("watching"),
		color.BlueString("%d", count),
		color.HiBlackString("files"))
}

func splitAndTrim(arg string) []string {
	var res []string
	if arg == "" {
		return res
	}
	split := strings.Split(arg, ",")
	for _, str := range split {
		res = append(res, strings.TrimSpace(str))
	}
	return res
}

func killCmd() {
	mu.Lock()
	if prev != nil {
		// flush any pending output
		cmdWriter.Flush()
		// send kill signal
		err := syscall.Kill(-prev.Process.Pid, syscall.SIGKILL)
		if err != nil {
			prettyWriter.WriteStringf("failed to kill prev running cmd: %s\n", err)
		}
	}
	mu.Unlock()
}

func executeCmd(cmd string) error {
	// kill prev process
	killCmd()

	// wait until ready
	<-ready

	// create command
	c := exec.Command("/bin/sh", "-c", cmd)

	// log cmd
	prettyWriter.WriteStringf("executing %s\n", color.MagentaString(cmd))

	// run command in another process
	f, err := pty.Start(c)
	if err != nil {
		return err
	}

	// proxy the output to the cmd writer
	cmdWriter.Proxy(f)

	// wait on process
	go func() {
		state, err := c.Process.Wait()

		// check exit code
		if stopOnNonZero && state.ExitCode() != 0 {
			if stopOnNonZero {
				prettyWriter.WriteStringf("exiting due to non-zero error code: %d\n", state.ExitCode())
				os.Exit(3)
			}
		}

		if err != nil {
			prettyWriter.WriteStringf("cmd encountered error: %s\n", err)
		}

		// clear prev
		mu.Lock()
		prev = nil
		mu.Unlock()
		// flag we are ready
		ready <- true
	}()

	// store process
	mu.Lock()
	prev = c
	mu.Unlock()
	return nil
}

func main() {

	watchStr := ""
	ignoreStr := ""

	flag.StringVar(&cmd, "cmd", "", "Shell command to run after detected changes")
	flag.StringVar(&watchStr, "watch", ".", "Comma separated file and directory globs to watch")
	flag.StringVar(&ignoreStr, "ignore", "", "Comma separated file and directory globs to ignore")
	flag.IntVar(&watchInterval, "interval", 400, "Watch scan interval, in milliseconds")
	flag.IntVar(&maxTokenSize, "max-token-size", 1024*1000*2, "Max output token size, in bytes")
	flag.BoolVar(&noSpinner, "no-spinner", false, "Disable fancy terminal spinner")
	flag.BoolVar(&stopOnNonZero, "stop-on-nonzero", false, "Stop witch process if the provided cmd returns a non-zero exit code")

	flag.Parse()

	if cmd == "" {
		os.Stderr.WriteString("No `--cmd` argument provided, Set command to execute with `--cmd=\"<shell command>\"`\n")
		os.Exit(1)
	}

	// watch targets are optional
	if watchStr == "" {
		os.Stderr.WriteString("No `--watch` arguments provided. Set watch targets with `--watch=\"<comma>,<separated>,<globs>...\"`\n")
		os.Exit(2)
	}
	watch = splitAndTrim(watchStr)

	// ignores are optional
	if ignoreStr != "" {
		ignore = splitAndTrim(ignoreStr)
	}

	// set token size
	cmdWriter.MaxTokenSize(maxTokenSize)

	// print logo
	fmt.Fprintf(os.Stdout, createLogo())

	// create the watcher
	w := watcher.New()

	// add watches
	for _, arg := range watch {
		prettyWriter.WriteStringf("watching %s\n", color.BlueString(arg))
		w.Watch(arg)
	}

	// add ignores first
	for _, arg := range ignore {
		prettyWriter.WriteStringf("ignoring %s\n", color.RedString(arg))
		w.Ignore(arg)
	}

	// check for initial target count
	numTargets, err := w.NumTargets()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to run initial scan: %s\n", err))
		os.Exit(3)
	}
	prettyWriter.WriteStringf("%s\n", fileCountString(numTargets))

	// gracefully shutdown cmd process on exit
	graceful.OnSignal(func() {
		// kill process
		prettyWriter.WriteStringf("\r") // hide the ^C
		killCmd()
		spin.Done()
		os.Exit(0)
	})

	// flag that we are ready to launch process
	ready <- true

	// launch cmd process
	err = executeCmd(cmd)
	if err != nil {
		prettyWriter.WriteStringf("failed to run cmd: %s\n", err)
	}

	// track which action to take
	nextWatch := watchInterval
	nextTick := tickInterval

	// start scan loop
	for {
		if nextWatch == watchInterval {
			// prev number targets
			prevTargets := numTargets

			// check if anything has changed
			events, err := w.ScanForEvents()
			if err != nil {
				prettyWriter.WriteStringf("failed to run scan: %s\n", err)
			}
			// log changes
			for _, event := range events {
				prettyWriter.WriteStringf("%s\n", fileChangeString(event.Path, event.Type))
				// update num targets
				if event.Type == watcher.Added {
					numTargets++
				}
				if event.Type == watcher.Removed {
					numTargets--
				}
			}

			// log new target count
			if prevTargets != numTargets {
				prettyWriter.WriteStringf("%s\n", fileCountString(numTargets))
			}

			// if so, execute command
			if len(events) > 0 {
				err := executeCmd(cmd)
				if err != nil {
					prettyWriter.WriteStringf("failed to run cmd: %s\n", err)
				}
			}
		}

		var sleep int

		if !noSpinner {
			// spinner enabled

			if nextTick == tickInterval {
				// spin ticker
				spin.Tick(numTargets)
			}

			if nextTick < nextWatch {
				// next iter is tick
				sleep = nextTick
				nextWatch -= nextTick
				// reset tick
				nextTick = tickInterval
			} else if nextTick > nextWatch {
				// next iter is watch
				sleep = nextWatch
				nextTick -= nextWatch
				// reset watch
				nextWatch = watchInterval
			} else {
				// next iter is iether
				sleep = nextTick
				// reset
				nextTick = tickInterval
				nextWatch = watchInterval
			}

		} else {
			// spinner disabled
			sleep = watchInterval
		}

		// sleep
		time.Sleep(time.Millisecond * time.Duration(sleep))
	}

}
