# witch

> Dead simple watching

[![Build Status](https://travis-ci.org/unchartedsoftware/witch.svg?branch=master)](https://travis-ci.org/unchartedsoftware/witch)
[![Go Report Card](https://goreportcard.com/badge/github.com/unchartedsoftware/witch)](https://goreportcard.com/report/github.com/unchartedsoftware/witch)

<img width="600" src="https://rawgit.com/unchartedsoftware/witch/master/screenshot.gif" alt="screenshot" />

## Description

Detects changes to files and directories then executes provided shell command. That's it.

## Features:
- Supports double-star globbing
- Detects files / directories added after starting watch
- Watch will persist if executed cmd fails (ex. compile / linting errors)
- Awesome magic wand terminal spinner

**Note**: Uses polling to work consistently across multiple platforms, therefore the CPU usage is dependent on the efficiently of your globs. With a minimal effort your watch should use less than 0.5% CPU. Will switch to event-based once [fsnofity](https://github.com/fsnotify/fsnotify) has matured sufficiently.

## Dependencies

Requires the [Go](https://golang.org/) programming language binaries with the `GOPATH` environment variable specified and `$GOPATH/bin` in your `PATH`.

## Installation

```bash
go get github.com/unchartedsoftware/witch
```

## Usage

```bash
witch --cmd=<shell-command> [--watch="<glob>,..."] [--ignore="<glob>,..."] [--interval=<milliseconds>]
```

Command-line args:

```
--cmd
	- Shell command to run after detected changes
--watch
	- Comma separated file and directory globs to watch (default: ".")
--ignore
	- Comma separated file and directory globs to ignore (default: "")
--interval
	- Watch scan interval, in milliseconds (default: 400)
--no-spinner
	- Disable fancy terminal spinner (default: false)
```

## Example

```bash
witch --cmd="make lint && make fmt && make run" --watch="main.go,api/**/*.go"
```
