# witch

> Dead simple watching

[![Build Status](https://travis-ci.org/unchartedsoftware/witch.svg?branch=master)](https://travis-ci.org/unchartedsoftware/witch)
[![Go Report Card](https://goreportcard.com/badge/github.com/unchartedsoftware/witch)](https://goreportcard.com/report/github.com/unchartedsoftware/witch)

<img width="600" src="https://rawgit.com/unchartedsoftware/witch/master/screenshot.png" alt="screenshot" />

## Description

Detect changes to files or directories and executes the provided command. Uses polling to work consistently across multiple platforms. Supports double-star globbing.

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
	- Comma separated file and directory globs to ignore (default: ".*")
--interval
	- Watch scan interval, in milliseconds (default: 400)
```

## Example

```bash
witch --cmd="make lint && make fmt && make run" --watch="main.go,api/**/*.go"
```
