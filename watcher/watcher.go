package watcher

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/pkg/errors"
)

const (
	// Added represents a file creation event.
	Added = "added"
	// Changed represents a file change event.
	Changed = "changed"
	// Removed represents a file removal event.
	Removed = "removed"
)

// Watcher represents a simple struct for scanning and checking for any changes
// that occur in a set of watched files and directories.
type Watcher struct {
	watches []string
	ignores []string
	prev    map[string]*Target
}

// Target represents a single watch target.
type Target struct {
	Path     string
	Fullpath string
	info     os.FileInfo
}

// Event represents a single detected file event.
type Event struct {
	Type   string
	Target *Target
}

// New instantiates and returns a new watcher struct.
func New() *Watcher {
	return &Watcher{}
}

// Watch adds a single file, directory, or glob to the file watch list.
func (w *Watcher) Watch(arg string) {
	w.watches = append(w.watches, arg)
}

// Ignore adds a single file, directory, or glob to the file ignore list.
func (w *Watcher) Ignore(arg string) {
	w.ignores = append(w.ignores, arg)
}

// ScanForEvents returns any events that occurred since the last scan.
func (w *Watcher) ScanForEvents() ([]*Event, error) {
	// get all current watches
	targets, err := w.getWatches()
	if err != nil {
		return nil, err
	}
	// check any events
	return w.check(targets), nil
}

// ScanForChange returns true if any events occurred since the last scan.
func (w *Watcher) ScanForChange() (bool, error) {
	// get all current watches
	targets, err := w.getWatches()
	if err != nil {
		return false, err
	}
	// check any events
	return w.checkBool(targets), nil
}

// NumTargets returns the number of currently watched targets.
func (w *Watcher) NumTargets() (uint64, error) {
	// get all current watches
	targets, err := w.getWatches()
	if err != nil {
		return 0, err
	}
	// scan for current status
	return uint64(len(targets)), nil
}

func (w *Watcher) expandArgs(args []string) (map[string]*Target, error) {
	results := make(map[string]*Target)
	for _, arg := range args {
		paths, err := doublestar.Glob(arg)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to expand glob %s", arg))
		}
		for _, path := range paths {
			fullpath, err := filepath.Abs(path)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("unable to get absolute path for %s", path))
			}
			results[fullpath] = &Target{
				Path:     path,
				Fullpath: fullpath,
			}
		}
	}
	return results, nil
}

func (w *Watcher) scan(targets map[string]*Target) (map[string]*Target, error) {
	results := make(map[string]*Target)
	for _, target := range targets {
		// make sure the path exists
		info, err := os.Stat(target.Fullpath)
		if err != nil {
			// can't find file
			continue
		}
		// if it's not a directory, skip to next path
		if !info.IsDir() {
			// append info
			target.info = info
			// add to map
			results[target.Fullpath] = target
			continue
		}
		// read directory contents
		infos, err := ioutil.ReadDir(target.Fullpath)
		if err != nil {
			return nil, errors.Wrap(err, "unable read dir")
		}
		// for each child
		subtargets := make(map[string]*Target)
		for _, info := range infos {
			// create sub-target
			fullpath := filepath.Join(target.Fullpath, info.Name())
			subtargets[fullpath] = &Target{
				Path:     filepath.Join(target.Path, info.Name()),
				Fullpath: fullpath,
			}
		}
		// scan children recursively
		children, err := w.scan(subtargets)
		if err != nil {
			return nil, err
		}
		// add to result
		for subpath, subtarget := range children {
			results[subpath] = subtarget
		}
	}
	return results, nil
}

func (w *Watcher) scanTargets(args []string) (map[string]*Target, error) {
	// expand args
	targets, err := w.expandArgs(args)
	if err != nil {
		return nil, err
	}
	// scan for all targets
	return w.scan(targets)
}

func (w *Watcher) getWatches() (map[string]*Target, error) {
	// expand watches
	watches, err := w.scanTargets(w.watches)
	if err != nil {
		return nil, err
	}
	// expand ignores
	ignores, err := w.scanTargets(w.ignores)
	if err != nil {
		return nil, err
	}
	// remove ignores from watches
	result := make(map[string]*Target)
	for fullpath, target := range watches {
		_, ok := ignores[fullpath]
		if !ok {
			result[fullpath] = target
		}
	}
	return result, nil
}

func (w *Watcher) check(latest map[string]*Target) []*Event {
	if w.prev == nil {
		w.prev = latest
		return nil
	}
	var events []*Event
	// for each current file, see if it is new, or has changed since prev scan
	for path, target := range latest {
		prev, ok := w.prev[path]
		if !ok {
			// new file
			events = append(events, &Event{
				Type:   Added,
				Target: target,
			})
		} else if !prev.info.ModTime().Equal(target.info.ModTime()) {
			// changed file
			events = append(events, &Event{
				Type:   Changed,
				Target: target,
			})
		}
		// remove from prev
		delete(w.prev, path)
	}
	// iterate over remaining prev files, as they no longer exist
	for _, target := range w.prev {
		// removed file
		events = append(events, &Event{
			Type:   Removed,
			Target: target,
		})
	}
	// store latest as prev for next iteration
	w.prev = latest
	return events
}

func (w *Watcher) checkBool(latest map[string]*Target) bool {
	if w.prev == nil {
		w.prev = latest
		return false
	}
	// for each current file, see if it is new, or has changed since prev scan
	for path, target := range latest {
		prev, ok := w.prev[path]
		if !ok {
			// new file
			w.prev = latest
			return true
		}
		if !prev.info.ModTime().Equal(target.info.ModTime()) {
			// changed file
			w.prev = latest
			return true
		}
		// remove from prev
		delete(w.prev, path)
	}
	// if remaining prev files, it means at least one file has been removed
	if len(w.prev) > 0 {
		// removed file
		w.prev = latest
		return true
	}
	w.prev = latest
	return false
}

func isSubDir(child, parent string) bool {
	rel, err := filepath.Rel(child, parent)
	if err != nil {
		return false
	}
	return filepath.HasPrefix(rel, "..")
}

func removeDuplicates(s []int) []int {
	seen := make(map[int]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
