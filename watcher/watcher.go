package watcher

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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

func (w *Watcher) expandIgnoreGlobs(args []string) ([]string, error) {
	var results []string
	for _, arg := range args {
		// trim unnecessary recursive expansion
		path := filepath.Clean(arg)
		path = strings.TrimSuffix(path, "**/*")
		// expand the glob
		paths, err := Glob(path, nil)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to expand glob %s", arg))
		}
		results = append(results, paths...)
	}
	return results, nil
}

func (w *Watcher) expandWatchGlobs(args []string, ignores map[string]bool) ([]string, error) {
	var results []string
	for _, arg := range args {
		// trim unnecessary recursive expansion
		path := filepath.Clean(arg)
		path = strings.TrimSuffix(path, "**/*")
		// expand the glob
		paths, err := Glob(path, ignores)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to expand glob %s", arg))
		}
		results = append(results, paths...)
	}
	return results, nil
}

func (w *Watcher) scanIgnores(args []string) (map[string]bool, error) {
	// expand args
	paths, err := w.expandIgnoreGlobs(args)
	if err != nil {
		return nil, err
	}
	// assemble map of ignores
	ignores := make(map[string]bool)
	for _, path := range paths {
		// get full path for target
		fullpath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to get absolute path for %s", path))
		}
		// make sure the path exists
		_, err = os.Stat(fullpath)
		if err != nil {
			// can't find file
			continue
		}
		// add to map
		ignores[path] = true
	}
	return ignores, nil
}

func (w *Watcher) scanWatches(paths []string, ignores map[string]bool) (map[string]*Target, error) {
	// gather watches that aren't ignored
	results := make(map[string]*Target)
	for _, path := range paths {
		// check if ignored
		if isIgnored(path, ignores) {
			continue
		}
		// get full path for target
		fullpath, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to get absolute path for %s", path))
		}
		// make sure the path exists
		info, err := os.Stat(fullpath)
		if err != nil {
			// can't find file
			continue
		}
		// if it's not a directory, skip to next path
		if !info.IsDir() {
			// add to map
			results[path] = &Target{
				Path:     path,
				Fullpath: fullpath,
				info:     info,
			}
			continue
		}
		// read directory contents
		infos, err := ioutil.ReadDir(fullpath)
		if err != nil {
			return nil, errors.Wrap(err, "unable read dir")
		}
		// for each child
		var subpaths []string
		for _, info := range infos {
			// create subpath
			subpaths = append(subpaths, filepath.Join(path, info.Name()))
		}
		// scan children recursively
		children, err := w.scanWatches(subpaths, ignores)
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

func (w *Watcher) scanWatchTargets(args []string, ignores map[string]bool) (map[string]*Target, error) {
	// expand args
	paths, err := w.expandWatchGlobs(args, ignores)
	if err != nil {
		return nil, err
	}
	// scan for all targets
	return w.scanWatches(paths, ignores)
}

func (w *Watcher) getWatches() (map[string]*Target, error) {
	// scan for ignores
	ignores, err := w.scanIgnores(w.ignores)
	if err != nil {
		return nil, err
	}
	// scan for watches
	return w.scanWatchTargets(w.watches, ignores)
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

func isIgnored(path string, ignores map[string]bool) bool {
	split := strings.Split(path, "/")
	accum := ""
	for _, str := range split {
		if accum != "" {
			accum += "/"
		}
		accum += str
		_, isIgnored := ignores[accum]
		if isIgnored {
			return true
		}
	}
	return false
}
