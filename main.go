package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/unchartedsoftware/witch/graceful"
	"github.com/unchartedsoftware/witch/watcher"
)

const (
	logo = `
        \    / o _|_  _ |_
         \/\/  |  |_ (_ | |`
)

var (
	watch    []string
	ignore   []string
	cmd      string
	interval int
	prev     *exec.Cmd
	mu       = &sync.Mutex{}
)

func writeToErr(format string, args ...interface{}) {
	stamp := color.BlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[witch]")
	//carot := color.MagentaString("★")
	msg := color.BlackString("- %s", fmt.Sprintf(format, args...))
	fmt.Fprintf(os.Stderr, "%s %s %s\n", stamp, name, msg)
}

func writeToOut(format string, args ...interface{}) {
	stamp := color.BlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[witch]")
	//carot := color.MagentaString("★")
	msg := color.BlackString("- %s", fmt.Sprintf(format, args...))
	fmt.Fprintf(os.Stdout, "%s %s %s\n", stamp, name, msg)
}

func changeString(path string, event string) string {
	star := color.MagentaString("★")
	msg := color.BlackString(fmt.Sprintf("%s %s", path, event))
	return fmt.Sprintf("%s %s %s  %s %s %s %s", star, star, star, msg, star, star, star)
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

func parseCLI() error {

	// define flags
	w := flag.String("watch", "", "Files and directories to watch")
	i := flag.String("ignore", "", "Files and directories  to ignore")
	c := flag.String("cmd", "", "Command to run after changes")
	in := flag.Int("interval", 400, "Watch interval in milliseconds")
	// parse the flags
	flag.Parse()
	// ensure we have watch targets
	if *w == "" {
		return fmt.Errorf("no `watch` targets provided, there must be at least one file / directory")
	}
	watch = splitAndTrim(*w)
	// ensure we have a command
	if *c == "" {
		return fmt.Errorf("no `cmd` string provided, what is the point of a watch if it does nothing?")
	}
	cmd = *c
	// ignores are optional
	if *i != "" {
		ignore = splitAndTrim(*i)
	}
	// set interval
	interval = *in
	return nil
}

func killCmd() {
	mu.Lock()
	if prev != nil {
		err := prev.Process.Kill()
		if err != nil {
			writeToErr("failed to kill prev running cmd: ", err)
		}
	}
	mu.Unlock()
}

func executeCmd(cmd string) error {
	// kill prev process
	killCmd()

	// lock
	mu.Lock()
	defer mu.Unlock()

	// create command
	c := exec.Command("sh", "-c", cmd)
	//c := exec.Command("/bin/sh", cmd)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	// log cmd
	writeToOut("executing %s", color.BlueString(cmd))

	// run command in another process
	err := c.Start()
	if err != nil {
		return err
	}

	// wait on process
	go func() {
		err := c.Wait()
		if err != nil {
			writeToErr("cmd encountered error: %s", err)
		}
		mu.Lock()
		prev = nil
		mu.Unlock()
	}()

	// store process
	prev = c
	return nil
}

func main() {
	// log logo
	fmt.Fprintf(os.Stdout, color.GreenString(logo))
	fmt.Fprintf(os.Stdout, "\n\n")

	// parse command line flags
	err := parseCLI()
	if err != nil {
		writeToErr("unable to parse flags:", err)
		os.Exit(1)
	}

	// create the watcher
	w := watcher.New()

	// add watches
	for _, arg := range watch {
		writeToOut("watching %s", color.MagentaString(arg))
		w.Watch(arg)
	}

	// add ignores first
	for _, arg := range ignore {
		writeToOut("ignoring %s", color.RedString(arg))
		w.Ignore(arg)
	}

	// gracefully shutdown cmd process on exit
	graceful.OnSignal(func() {
		// kill process
		killCmd()
		os.Exit(0)
	})

	// launch cmd process
	executeCmd(cmd)

	// start scan loop
	for {
		// check if anything has changed
		events, err := w.ScanForEvents()
		if err != nil {
			writeToErr("failed to run scan: %s", err)
		}
		// log changes
		for _, event := range events {
			writeToOut(changeString(event.Info.Name(), event.Type))
		}
		// if so, execute command
		if len(events) > 0 {
			err := executeCmd(cmd)
			if err != nil {
				writeToErr("failed to run cmd: %s", err)
			}
		}
		// sleep
		time.Sleep(time.Millisecond * time.Duration(interval))
	}
}
