package main

import (
	"reflect"
	"testing"

	"github.com/fatih/color"

	"github.com/kbirk/witch/watcher"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want []string
	}{
		{
			name: "empty",
			arg:  "",
			want: nil,
		},
		{
			name: "single value",
			arg:  "  main.go  ",
			want: []string{"main.go"},
		},
		{
			name: "comma separated values",
			arg:  "main.go, api/**/*.go ,README.md",
			want: []string{"main.go", "api/**/*.go", "README.md"},
		},
		{
			name: "preserves empty segments",
			arg:  "main.go,, README.md, ",
			want: []string{"main.go", "", "README.md", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.arg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitAndTrim(%q) = %#v, want %#v", tt.arg, got, tt.want)
			}
		})
	}
}

func TestFileCountString(t *testing.T) {
	withNoColor(t)

	tests := []struct {
		count uint64
		want  string
	}{
		{count: 0, want: "no files found"},
		{count: 1, want: "watching 1 file"},
		{count: 2, want: "watching 2 files"},
	}

	for _, tt := range tests {
		got := fileCountString(tt.count)
		if got != tt.want {
			t.Fatalf("fileCountString(%d) = %q, want %q", tt.count, got, tt.want)
		}
	}
}

func TestFileChangeString(t *testing.T) {
	withNoColor(t)

	tests := []struct {
		event watcher.EventType
		want  string
	}{
		{event: watcher.Added, want: "src/main.go added"},
		{event: watcher.Changed, want: "src/main.go changed"},
		{event: watcher.Removed, want: "src/main.go removed"},
		{event: watcher.EventType(99), want: "src/main.go changed"},
	}

	for _, tt := range tests {
		got := fileChangeString("src/main.go", tt.event)
		if got != tt.want {
			t.Fatalf("fileChangeString(%d) = %q, want %q", tt.event, got, tt.want)
		}
	}
}

func withNoColor(t *testing.T) {
	t.Helper()
	oldNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = oldNoColor
	})
}
