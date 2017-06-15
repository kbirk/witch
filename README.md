# witch

> Dead simple watching

[![Build Status](https://travis-ci.org/unchartedsoftware/witch.svg?branch=master)](https://travis-ci.org/unchartedsoftware/witch)
[![Go Report Card](https://goreportcard.com/badge/github.com/unchartedsoftware/witch)](https://goreportcard.com/report/github.com/unchartedsoftware/witch)

## Description

Detect changes to files or directories and executes the provided command. Uses polling to consistently work across multiple platforms. Supports globbing.

## Dependencies

Requires the [Go](https://golang.org/) programming language binaries with the `GOPATH` environment variable specified and `$GOPATH/bin` in your `PATH`.

## Installation

```bash
go get github.com/unchartedsoftware/witch
```

## Usage

Command-line args:

```
-watch
	comma separated globs to watch.
-ignore
	comma separated globs to ignore.
-interval
	watcher poll interval in milliseconds (default=400).
-cmd
	shell command to execute on startup and changes.
```

## Example

```bash
witch -watch="./**/*.go" -ignore="vendor" -cmd="make lint && make fmt && make run"
```
