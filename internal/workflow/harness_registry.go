package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type harnessRegistry struct {
	rootDir    string
	logger     *slog.Logger
	watcher    *fsnotify.Watcher
	onReload   func(harnessReloadEvent)
	mu         sync.RWMutex
	cache      map[string]cachedHarness
	pending    map[string]string
	watchedDir map[string]struct{}
	done       chan struct{}
}

type cachedHarness struct {
	content string
	hash    string
}

type harnessReloadEvent struct {
	RelativePath    string
	PreviousContent string
	CurrentContent  string
}

func newHarnessRegistry(rootDir string, logger *slog.Logger, onReload func(harnessReloadEvent)) (*harnessRegistry, error) {
	if err := os.MkdirAll(rootDir, 0o750); err != nil {
		return nil, fmt.Errorf("create harness root: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create harness watcher: %w", err)
	}

	registry := &harnessRegistry{
		rootDir:    rootDir,
		logger:     logger,
		watcher:    watcher,
		onReload:   onReload,
		cache:      map[string]cachedHarness{},
		pending:    map[string]string{},
		watchedDir: map[string]struct{}{},
		done:       make(chan struct{}),
	}

	if err := registry.addDirWatchRecursive(rootDir); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	go registry.run()

	return registry, nil
}

func (r *harnessRegistry) Close() error {
	if r == nil || r.watcher == nil {
		return nil
	}

	if err := r.watcher.Close(); err != nil {
		return err
	}

	<-r.done
	return nil
}

func (r *harnessRegistry) Exists(relativePath string) bool {
	_, err := os.Stat(r.absolutePath(relativePath))
	return err == nil
}

func (r *harnessRegistry) Read(relativePath string) (string, error) {
	r.mu.RLock()
	entry, ok := r.cache[relativePath]
	r.mu.RUnlock()
	if ok {
		return entry.content, nil
	}

	content, hash, err := r.readFile(r.absolutePath(relativePath))
	if err != nil {
		return "", err
	}

	r.mu.Lock()
	r.cache[relativePath] = cachedHarness{content: content, hash: hash}
	r.mu.Unlock()

	return content, nil
}

func (r *harnessRegistry) Write(relativePath string, content string) error {
	absolutePath := r.absolutePath(relativePath)
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		return fmt.Errorf("create harness parent directory: %w", err)
	}
	if err := r.addDirWatchRecursive(filepath.Dir(absolutePath)); err != nil {
		return err
	}

	hash := hashContent(content)

	r.mu.Lock()
	r.pending[absolutePath] = hash
	r.cache[relativePath] = cachedHarness{content: content, hash: hash}
	r.mu.Unlock()

	if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
		r.mu.Lock()
		delete(r.pending, absolutePath)
		delete(r.cache, relativePath)
		r.mu.Unlock()
		return fmt.Errorf("write harness file: %w", err)
	}

	return nil
}

func (r *harnessRegistry) Delete(relativePath string) error {
	absolutePath := r.absolutePath(relativePath)

	r.mu.Lock()
	delete(r.pending, absolutePath)
	delete(r.cache, relativePath)
	r.mu.Unlock()

	err := os.Remove(absolutePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete harness file: %w", err)
	}

	return nil
}

func (r *harnessRegistry) absolutePath(relativePath string) string {
	return filepath.Join(r.rootDir, filepath.FromSlash(strings.TrimPrefix(relativePath, ".openase/harnesses/")))
}

func (r *harnessRegistry) relativePath(absolutePath string) (string, bool) {
	relative, err := filepath.Rel(r.rootDir, absolutePath)
	if err != nil {
		return "", false
	}
	if strings.HasPrefix(relative, "..") {
		return "", false
	}

	return filepath.ToSlash(filepath.Join(".openase", "harnesses", relative)), true
}

func (r *harnessRegistry) readFile(absolutePath string) (string, string, error) {
	//nolint:gosec // harness paths are resolved relative to the validated registry root
	contentBytes, err := os.ReadFile(absolutePath)
	if err != nil {
		return "", "", fmt.Errorf("read harness file: %w", err)
	}

	content := string(contentBytes)
	return content, hashContent(content), nil
}

func (r *harnessRegistry) addDirWatchRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk harness directory %s: %w", root, err)
		}
		if !entry.IsDir() {
			return nil
		}

		return r.addDirWatch(path)
	})
}

func (r *harnessRegistry) addDirWatch(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.watchedDir[path]; ok {
		return nil
	}

	if err := r.watcher.Add(path); err != nil {
		return fmt.Errorf("watch harness directory %s: %w", path, err)
	}
	r.watchedDir[path] = struct{}{}

	return nil
}

func (r *harnessRegistry) run() {
	defer close(r.done)

	for {
		select {
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}
			r.handleEvent(event)
		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			r.logger.Error("harness watcher error", "error", err)
		}
	}
}

func (r *harnessRegistry) handleEvent(event fsnotify.Event) {
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 && event.Op&fsnotify.Remove == 0 {
		return
	}

	info, statErr := os.Stat(event.Name)
	if statErr == nil && info.IsDir() {
		if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
			if err := r.addDirWatchRecursive(event.Name); err != nil {
				r.logger.Error("watch new harness directory", "error", err, "path", event.Name)
			}
		}
		return
	}

	relativePath, ok := r.relativePath(event.Name)
	if !ok {
		return
	}

	if event.Op&fsnotify.Remove != 0 {
		r.mu.Lock()
		delete(r.pending, event.Name)
		delete(r.cache, relativePath)
		r.mu.Unlock()
		return
	}

	content, hash, err := r.readFile(event.Name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			r.mu.Lock()
			delete(r.pending, event.Name)
			delete(r.cache, relativePath)
			r.mu.Unlock()
			return
		}

		r.logger.Error("read harness after change", "error", err, "path", event.Name)
		return
	}

	shouldReload := false

	r.mu.Lock()
	previous := r.cache[relativePath]
	if pendingHash, ok := r.pending[event.Name]; ok && pendingHash == hash {
		delete(r.pending, event.Name)
		r.cache[relativePath] = cachedHarness{content: content, hash: hash}
		r.mu.Unlock()
		return
	}
	r.cache[relativePath] = cachedHarness{content: content, hash: hash}
	shouldReload = previous.hash != "" && previous.hash != hash
	if previous.hash == "" {
		shouldReload = true
	}
	r.mu.Unlock()

	if shouldReload && r.onReload != nil {
		r.onReload(harnessReloadEvent{
			RelativePath:    relativePath,
			PreviousContent: previous.content,
			CurrentContent:  content,
		})
	}
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
