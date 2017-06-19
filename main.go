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
	"github.com/unchartedsoftware/witch/spinner"
	"github.com/unchartedsoftware/witch/watcher"
	"github.com/unchartedsoftware/witch/writer"
)

const (
	name    = "witch"
	version = "0.2.4"
)

var (
	watch         []string
	ignore        []string
	cmd           string
	watchInterval int
	noSpinner     bool
	tickInterval  = 100
	prev          *exec.Cmd
	ready         = make(chan bool, 1)
	mu            = &sync.Mutex{}
	prettyOut     = writer.NewPretty(name, os.Stdout)
	prettyErr     = writer.NewPretty(name, os.Stderr)
	cmdOut        = writer.NewCmd(os.Stdout)
	cmdErr        = writer.NewCmd(os.Stderr)
	spin          = spinner.New(prettyOut)
)

func createLogo() string {
	return color.GreenString("\n        \\    / ") +
		color.MagentaString("★") +
		color.GreenString(" _|_  _ |_\n         ") +
		color.GreenString("\\/\\/  |  |_ |_ | |\n\n        ") +
		color.BlackString("version %s\n\n", version)
}

func fileChangeString(path string, event string) string {
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

func fileCountString(count uint64) string {
	switch count {
	case 0:
		return color.BlackString("no files found")
	case 1:
		return fmt.Sprintf("%s %s %s",
			color.BlackString("watching"),
			color.BlueString("%d", count),
			color.BlackString("file"))
	}
	return fmt.Sprintf("%s %s %s",
		color.BlackString("watching"),
		color.BlueString("%d", count),
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
			prettyErr.WriteStringf("failed to kill prev running cmd: %s\n", err)
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
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Stdin = os.Stdin
	c.Stdout = cmdOut
	c.Stderr = cmdErr

	// log cmd
	prettyOut.WriteStringf("executing %s\n", color.MagentaString(cmd))

	// run command in another process
	err := c.Start()
	if err != nil {
		return err
	}

	// wait on process
	go func() {
		_, err := c.Process.Wait()
		if err != nil {
			prettyErr.WriteStringf("cmd encountered error: %s\n", err)
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
	app := cli.NewApp()
	app.Name = name
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
			Value: "",
			Usage: "Comma separated file and directory globs to ignore",
		},
		cli.IntFlag{
			Name:  "interval",
			Value: 400,
			Usage: "Watch scan interval, in milliseconds",
		},
		cli.BoolFlag{
			Name:  "no-spinner",
			Usage: "Disable fancy terminal spinner",
		},
	}
	app.Action = func(c *cli.Context) error {

		// validate command line flags

		// ensure we have a command
		if c.String("cmd") == "" {
			return cli.NewExitError("No `--cmd` argument provided, Set command to execute with `--cmd=\"<shell command>\"`", 1)
		}
		cmd = c.String("cmd")

		// watch targets are optional
		if c.String("watch") == "" {
			return cli.NewExitError("No `--watch` arguments provided. Set watch targets with `--watch=\"<comma>,<separated>,<globs>...\"`", 2)
		}
		watch = splitAndTrim(c.String("watch"))

		// ignores are optional
		if c.String("ignore") != "" {
			ignore = splitAndTrim(c.String("ignore"))
		}

		// watchInterval is optional
		watchInterval = c.Int("interval")

		// disable spinner
		noSpinner = c.Bool("no-spinner")

		// print logo
		fmt.Fprintf(os.Stdout, createLogo())

		// create the watcher
		w := watcher.New()

		// add watches
		for _, arg := range watch {
			prettyOut.WriteStringf("watching %s\n", color.BlueString(arg))
			w.Watch(arg)
		}

		// add ignores first
		for _, arg := range ignore {
			prettyOut.WriteStringf("ignoring %s\n", color.RedString(arg))
			w.Ignore(arg)
		}

		// check for initial target count
		numTargets, err := w.NumTargets()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Failed to run initial scan: %s", err), 3)
		}
		prettyOut.WriteStringf("%s\n", fileCountString(numTargets))

		// gracefully shutdown cmd process on exit
		graceful.OnSignal(func() {
			// kill process
			killCmd()
			spin.Done()
			os.Exit(0)
		})

		// flag that we are ready to launch process
		ready <- true

		// launch cmd process
		executeCmd(cmd)

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
					prettyErr.WriteStringf("failed to run scan: %s\n", err)
				}
				// log changes
				for _, event := range events {
					prettyOut.WriteStringf("%s\n", fileChangeString(event.Path, event.Type))
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
					prettyOut.WriteStringf("%s\n", fileCountString(numTargets))
				}

				// if so, execute command
				if len(events) > 0 {
					err := executeCmd(cmd)
					if err != nil {
						prettyErr.WriteStringf("failed to run cmd: %s\n", err)
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
	// run app
	app.Run(os.Args)
}
