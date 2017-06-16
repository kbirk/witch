package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/urfave/cli"

	"github.com/unchartedsoftware/witch/graceful"
	"github.com/unchartedsoftware/witch/watcher"
)

const (
	version = "0.1.1"
)

var (
	watch    []string
	ignore   []string
	cmd      string
	interval int
	prev     *exec.Cmd
	ready    = make(chan bool, 1)
	mu       = &sync.Mutex{}
)

func createLogo() string {
	return color.GreenString("\n        \\    / ") +
		color.MagentaString("★") +
		color.GreenString(" _|_  _ |_\n         ") +
		color.GreenString("\\/\\/  |  |_ |_ | |\n\n        ") +
		color.BlackString("version "+version+"\n\n")
}

func writeToErr(format string, args ...interface{}) {
	stamp := color.BlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[witch]")
	msg := color.BlackString("- %s", fmt.Sprintf(format, args...))
	fmt.Fprintf(os.Stderr, "%s %s %s\n", stamp, name, msg)
}

func writeToOut(format string, args ...interface{}) {
	stamp := color.BlackString("[%s]", time.Now().Format(time.Stamp))
	name := color.GreenString("[witch]")
	msg := color.BlackString("- %s", fmt.Sprintf(format, args...))
	fmt.Fprintf(os.Stdout, "%s %s %s\n", stamp, name, msg)
}

func changeString(path string, event string) string {
	switch event {
	case watcher.Added:
		return fmt.Sprintf("%s %s",
			color.BlackString(path),
			color.GreenString(event))
	case watcher.Removed:
		return fmt.Sprintf("%s %s",
			color.BlackString(path),
			color.RedString(event))
	}
	return fmt.Sprintf("%s %s",
		color.BlackString(path),
		color.BlueString(event))
}

func countString(count uint64) string {
	switch count {
	case 0:
		return color.BlackString("no files found")
	case 1:
		return fmt.Sprintf("%s %s %s",
			color.BlackString("watching"),
			color.CyanString(fmt.Sprintf("%d", count)),
			color.BlackString("file"))
	}
	return fmt.Sprintf("%s %s %s",
		color.BlackString("watching"),
		color.CyanString(fmt.Sprintf("%d", count)),
		color.BlackString("files"))

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
		err := syscall.Kill(-prev.Process.Pid, syscall.SIGKILL)
		if err != nil {
			writeToErr("failed to kill prev running cmd: ", err)
		}
		prev = nil
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
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	// log cmd
	writeToOut("executing %s", color.MagentaString(cmd))

	// run command in another process
	err := c.Start()
	if err != nil {
		return err
	}

	// wait on process
	go func() {
		_, err := c.Process.Wait()
		if err != nil {
			writeToErr("cmd encountered error: %s", err)
		}
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
	app := cli.NewApp()
	app.Name = "witch"
	app.Version = version
	app.Usage = "Dead simple watching"
	app.UsageText = "witch --cmd=<shell-command> [--watch=\"<glob>,...\"] [--ignore=\"<glob>,...\"] [--interval=<milliseconds>]"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "cmd",
			Value: "",
			Usage: "Shell command to run after detected changes",
		},
		cli.StringFlag{
			Name:  "watch",
			Value: ".",
			Usage: "Comma separated file and directory globs to watch",
		},
		cli.StringFlag{
			Name:  "ignore",
			Value: ".*",
			Usage: "Comma separated file and directory globs to ignore",
		},
		cli.IntFlag{
			Name:  "interval",
			Value: 400,
			Usage: "Watch scan interval, in milliseconds",
		},
	}
	app.Action = func(c *cli.Context) error {

		// validate command line flags

		// ensure we have a command
		if c.String("cmd") == "" {
			return cli.NewExitError("No `--cmd` argument provided, Set command to execute with `--cmd=\"<shell command>\"`", 2)
		}
		cmd = c.String("cmd")

		// watch targets are optional
		if c.String("watch") == "" {
			return cli.NewExitError("No `--watch` arguments provided. Set watch targets with `--watch=\"<comma>,<separated>,<globs>...\"`", 1)
		}
		watch = splitAndTrim(c.String("watch"))

		// ignores are optional
		if c.String("ignore") != "" {
			ignore = splitAndTrim(c.String("ignore"))
		}

		// interval is optional
		interval = c.Int("interval")

		// print logo
		fmt.Fprintf(os.Stdout, createLogo())

		// create the watcher
		w := watcher.New()

		// add watches
		for _, arg := range watch {
			writeToOut("watching %s", color.BlueString(arg))
			w.Watch(arg)
		}

		// add ignores first
		for _, arg := range ignore {
			writeToOut("ignoring %s", color.RedString(arg))
			w.Ignore(arg)
		}

		// check for initial target count
		numTargets, err := w.NumTargets()
		if err != nil {
			writeToErr("failed to run scan: %s", err)
		}
		writeToOut(countString(numTargets))

		// gracefully shutdown cmd process on exit
		graceful.OnSignal(func() {
			// kill process
			killCmd()
			os.Exit(0)
		})

		// flag that we are ready to launch process
		ready <- true

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
			prevTargets := numTargets
			for _, event := range events {
				writeToOut(changeString(event.Target.Path, event.Type))
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
				writeToOut(countString(numTargets))
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
	// run app
	app.Run(os.Args)
}
