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
	prev    map[string]os.FileInfo
}

// Event represents a single detected file event.
type Event struct {
	Type string
	Info os.FileInfo
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
func (w *Watcher) ScanForEvents() ([]Event, error) {
	return w.scanAndCheck()
}

// ScanForChange returns true if any events occurred since the last scan.
func (w *Watcher) ScanForChange() (bool, error) {
	return w.scanAndCheckBool()
}

func (w *Watcher) expandArgs(args []string) (map[string]bool, error) {
	results := make(map[string]bool)
	for _, arg := range args {
		expansion, err := filepath.Glob(arg)
		if err != nil {
			return nil, errors.Wrap(err, "unable to expand glob")
		}
		for _, expanded := range expansion {
			fullpath, err := filepath.Abs(expanded)
			if err != nil {
				return nil, errors.Wrap(err, "unable to get absolute path")
			}
			results[fullpath] = true
		}
	}
	return results, nil
}

func (w *Watcher) getWatches() ([]string, error) {
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
	var result []string
	for fullpath := range watches {
		_, ok := ignores[fullpath]
		if !ok {
			result = append(result, fullpath)
		}
	}
	return result, nil
}

func (w *Watcher) scan(paths []string) (map[string]os.FileInfo, error) {
	result := make(map[string]os.FileInfo)
	for _, path := range paths {
		// make sure the path exists
		info, err := os.Stat(path)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find watched file")
		}
		result[path] = info
		// if it's not a directory, skip to next path
		if !info.IsDir() {
			continue
		}
		// read directory contents
		infos, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, errors.Wrap(err, "unable read dir")
		}
		// for each child
		var subpaths []string
		for _, info := range infos {
			subpaths = append(subpaths, filepath.Join(path, info.Name()))
		}
		// get children recursively
		children, err := w.scan(subpaths)
		if err != nil {
			return nil, err
		}
		// add to result
		for subpath, info := range children {
			result[subpath] = info
		}
	}
	return result, nil
}

func (w *Watcher) check(latest map[string]os.FileInfo) []Event {
	if w.prev == nil {
		w.prev = latest
		return nil
	}
	var events []Event
	// for each current file, see if it is new, or has changed since prev scan
	for path, info := range latest {
		prev, ok := w.prev[path]
		if !ok {
			// new file
			events = append(events, Event{
				Type: Added,
				Info: info,
			})
		} else if !prev.ModTime().Equal(info.ModTime()) {
			// changed file
			events = append(events, Event{
				Type: Changed,
				Info: info,
			})
		}
		// remove from prev
		delete(w.prev, path)
	}
	// iterate over remaining prev files, as they no longer exist
	for _, info := range w.prev {
		// removed file
		events = append(events, Event{
			Type: Removed,
			Info: info,
		})
	}
	// store latest as prev for next iteration
	w.prev = latest
	return events
}

func (w *Watcher) checkBool(latest map[string]os.FileInfo) bool {
	if w.prev == nil {
		w.prev = latest
		return false
	}
	// for each current file, see if it is new, or has changed since prev scan
	for path, info := range latest {
		prev, ok := w.prev[path]
		if !ok {
			// new file
			w.prev = latest
			return true
		}
		if !prev.ModTime().Equal(info.ModTime()) {
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

func (w *Watcher) scanAndCheck() ([]Event, error) {
	// get all current watches
	watches, err := w.getWatches()
	if err != nil {
		return nil, err
	}
	// scan for current status
	infos, err := w.scan(watches)
	if err != nil {
		return nil, err
	}
	// check any events
	return w.check(infos), nil
}

func (w *Watcher) scanAndCheckBool() (bool, error) {
	// get all current watches
	watches, err := w.getWatches()
	if err != nil {
		return false, err
	}
	// scan for current status
	infos, err := w.scan(watches)
	if err != nil {
		return false, err
	}
	// check any events
	return w.checkBool(infos), nil
}
