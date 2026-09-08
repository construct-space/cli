package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	Path string
	Op   fsnotify.Op
}

type Watcher struct {
	watcher  *fsnotify.Watcher
	Events   chan Event
	Errors   chan error
	debounce time.Duration
	done     chan struct{}
}

func New(debounce time.Duration) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  fsw,
		Events:   make(chan Event, 100),
		Errors:   make(chan error, 10),
		debounce: debounce,
		done:     make(chan struct{}),
	}

	go w.loop()
	return w, nil
}

func (w *Watcher) AddRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "dist" {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) Add(path string) error {
	return w.watcher.Add(path)
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}

func (w *Watcher) loop() {
	var mu sync.Mutex
	pending := make(map[string]*time.Timer)

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			mu.Lock()
			if t, exists := pending[event.Name]; exists {
				t.Stop()
			}
			pending[event.Name] = time.AfterFunc(w.debounce, func() {
				w.Events <- Event{Path: event.Name, Op: event.Op}
				mu.Lock()
				delete(pending, event.Name)
				mu.Unlock()
			})
			mu.Unlock()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.Errors <- err
		}
	}
}
