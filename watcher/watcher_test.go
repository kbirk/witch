package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanForEventsDetectsAddChangeRemove(t *testing.T) {
	root := t.TempDir()
	existingPath := filepath.Join(root, "existing.txt")
	writeFileAt(t, existingPath, "existing", time.Unix(1700000000, 0))

	w := New()
	w.Watch(root)

	events, err := w.ScanForEvents()
	if err != nil {
		t.Fatalf("initial scan failed: %v", err)
	}
	assertNoEvents(t, events)

	addedPath := filepath.Join(root, "added.txt")
	writeFileAt(t, addedPath, "added", time.Unix(1700000100, 0))
	assertEvents(t, scanEvents(t, w), Event{Type: Added, Path: addedPath})

	writeFileAt(t, addedPath, "changed", time.Unix(1700000200, 0))
	assertEvents(t, scanEvents(t, w), Event{Type: Changed, Path: addedPath})

	if err := os.Remove(addedPath); err != nil {
		t.Fatalf("failed to remove %s: %v", addedPath, err)
	}
	assertEvents(t, scanEvents(t, w), Event{Type: Removed, Path: addedPath})
}

func TestWatcherIgnoresSubtrees(t *testing.T) {
	root := t.TempDir()
	keepPath := filepath.Join(root, "keep.txt")
	ignoredDir := filepath.Join(root, "ignored")
	ignoredPath := filepath.Join(ignoredDir, "hidden.txt")

	writeFileAt(t, keepPath, "keep", time.Unix(1700000000, 0))
	writeFileAt(t, ignoredPath, "hidden", time.Unix(1700000000, 0))

	w := New()
	w.Watch(root)
	w.Ignore(ignoredDir)

	count, err := w.NumTargets()
	if err != nil {
		t.Fatalf("counting targets failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("NumTargets() = %d, want 1", count)
	}
	assertNoEvents(t, scanEvents(t, w))

	writeFileAt(t, ignoredPath, "hidden changed", time.Unix(1700000100, 0))
	writeFileAt(t, filepath.Join(ignoredDir, "new.txt"), "new hidden", time.Unix(1700000100, 0))
	assertNoEvents(t, scanEvents(t, w))

	writeFileAt(t, keepPath, "keep changed", time.Unix(1700000200, 0))
	assertEvents(t, scanEvents(t, w), Event{Type: Changed, Path: keepPath})
}

func TestWatcherGlobFiltersTargets(t *testing.T) {
	root := t.TempDir()
	goPath := filepath.Join(root, "main.go")
	markdownPath := filepath.Join(root, "README.md")

	w := New()
	w.Watch(filepath.Join(root, "*.go"))
	assertNoEvents(t, scanEvents(t, w))

	writeFileAt(t, markdownPath, "# docs", time.Unix(1700000000, 0))
	assertNoEvents(t, scanEvents(t, w))

	writeFileAt(t, goPath, "package main\n", time.Unix(1700000100, 0))
	assertEvents(t, scanEvents(t, w), Event{Type: Added, Path: goPath})
}

func scanEvents(t *testing.T, w *Watcher) []Event {
	t.Helper()
	events, err := w.ScanForEvents()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	return events
}

func writeFileAt(t *testing.T, path, contents string, modTime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("failed to set modtime for %s: %v", path, err)
	}
}

func assertNoEvents(t *testing.T, events []Event) {
	t.Helper()
	if len(events) != 0 {
		t.Fatalf("events = %#v, want none", events)
	}
}

func assertEvents(t *testing.T, events []Event, expected ...Event) {
	t.Helper()
	if len(events) != len(expected) {
		t.Fatalf("events = %#v, want %#v", events, expected)
	}

	remaining := make(map[Event]bool, len(events))
	for _, event := range events {
		remaining[event] = true
	}
	for _, event := range expected {
		if !remaining[event] {
			t.Fatalf("events = %#v, missing %#v", events, event)
		}
	}
}
