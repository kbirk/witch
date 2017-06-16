package watcher

import (
	"io/ioutil"
	"os"
	"path/filepath"

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

// Op describe what type of event has occurred during the watching process.
type Op uint32

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
	watches, err := w.getWatches()
	if err != nil {
		return nil, err
	}
	// scan for current targets
	targets, err := w.scan(watches)
	if err != nil {
		return nil, err
	}
	// check any events
	return w.check(targets), nil
}

// ScanForChange returns true if any events occurred since the last scan.
func (w *Watcher) ScanForChange() (bool, error) {
	// get all current watches
	watches, err := w.getWatches()
	if err != nil {
		return false, err
	}
	// scan for current targets
	targets, err := w.scan(watches)
	if err != nil {
		return false, err
	}
	// check any events
	return w.checkBool(targets), nil
}

// NumTargets returns the number of currently watched targets.
func (w *Watcher) NumTargets() (uint64, error) {
	// get all current watches
	watches, err := w.getWatches()
	if err != nil {
		return 0, err
	}
	// scan for current targets
	targets, err := w.scan(watches)
	if err != nil {
		return 0, err
	}
	// scan for current status
	return uint64(len(targets)), nil
}

func (w *Watcher) expandArgs(args []string) (map[string]*Target, error) {
	results := make(map[string]*Target)
	for _, arg := range args {
		paths, err := filepath.Glob(arg)
		if err != nil {
			return nil, errors.Wrap(err, "unable to expand glob")
		}
		for _, path := range paths {
			fullpath, err := filepath.Abs(path)
			if err != nil {
				return nil, errors.Wrap(err, "unable to get absolute path")
			}
			results[fullpath] = &Target{
				Path:     path,
				Fullpath: fullpath,
			}
		}
	}
	return results, nil
}

func (w *Watcher) getWatches() ([]*Target, error) {
	// expand watches
	watches, err := w.expandArgs(w.watches)
	if err != nil {
		return nil, errors.Wrap(err, "unable to expand watched arguments")
	}
	// expand ignores
	ignores, err := w.expandArgs(w.ignores)
	if err != nil {
		return nil, errors.Wrap(err, "unable to expand ignored arguments")
	}
	// remove ignores from watches
	var result []*Target
	for fullpath, target := range watches {
		_, ok := ignores[fullpath]
		if !ok {
			result = append(result, target)
		}
	}
	return result, nil
}

func (w *Watcher) scan(targets []*Target) (map[string]*Target, error) {
	result := make(map[string]*Target)
	for _, target := range targets {
		// make sure the path exists
		info, err := os.Stat(target.Fullpath)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find watched file")
		}
		// if it's not a directory, skip to next path
		if !info.IsDir() {
			// append info
			target.info = info
			// add to map
			result[target.Fullpath] = target
			continue
		}
		// read directory contents
		infos, err := ioutil.ReadDir(target.Fullpath)
		if err != nil {
			return nil, errors.Wrap(err, "unable read dir")
		}
		// for each child
		var subtargets []*Target
		for _, info := range infos {
			// create sub-target
			subtarget := &Target{
				Path:     filepath.Join(target.Path, info.Name()),
				Fullpath: filepath.Join(target.Fullpath, info.Name()),
			}
			subtargets = append(subtargets, subtarget)
		}
		// scan children recursively
		children, err := w.scan(subtargets)
		if err != nil {
			return nil, err
		}
		// add to result
		for subpath, subtarget := range children {
			result[subpath] = subtarget
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
