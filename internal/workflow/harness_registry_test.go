package workflow

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestHarnessRegistryIgnoresPartialPendingWriteEvents(t *testing.T) {
	t.Parallel()

	registry := &harnessRegistry{
		rootDir:    t.TempDir(),
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		cache:      map[string]cachedHarness{},
		pending:    map[string]string{},
		watchedDir: map[string]struct{}{},
	}

	relativePath := ".openase/harnesses/project/coding.md"
	absolutePath := registry.absolutePath(relativePath)
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}

	previousContent := "---\nworkflow:\n  role: coding\n---\n\n# Coding\n"
	finalContent := "---\nworkflow:\n  role: coding\nskills:\n  - commit\n---\n\n# Coding\n"
	registry.cache[relativePath] = cachedHarness{
		content: previousContent,
		hash:    hashContent(previousContent),
	}
	registry.pending[absolutePath] = hashContent(finalContent)

	reloadCalls := 0
	registry.onReload = func(harnessReloadEvent) {
		reloadCalls++
	}

	if err := os.WriteFile(absolutePath, []byte("---\nworkflow:\n  role"), 0o600); err != nil {
		t.Fatalf("write partial harness: %v", err)
	}
	registry.handleEvent(fsnotify.Event{Name: absolutePath, Op: fsnotify.Write})

	got, err := registry.Read(relativePath)
	if err != nil {
		t.Fatalf("read cached harness after partial write: %v", err)
	}
	if got != previousContent {
		t.Fatalf("cached harness after partial write = %q, want %q", got, previousContent)
	}
	if reloadCalls != 0 {
		t.Fatalf("reloadCalls after partial write = %d, want 0", reloadCalls)
	}
	if _, ok := registry.pending[absolutePath]; !ok {
		t.Fatal("expected pending write to remain until final content arrives")
	}

	if err := os.WriteFile(absolutePath, []byte(finalContent), 0o600); err != nil {
		t.Fatalf("write final harness: %v", err)
	}
	registry.handleEvent(fsnotify.Event{Name: absolutePath, Op: fsnotify.Write})

	got, err = registry.Read(relativePath)
	if err != nil {
		t.Fatalf("read cached harness after final write: %v", err)
	}
	if got != finalContent {
		t.Fatalf("cached harness after final write = %q, want %q", got, finalContent)
	}
	if reloadCalls != 0 {
		t.Fatalf("reloadCalls after final write = %d, want 0", reloadCalls)
	}
	if _, ok := registry.pending[absolutePath]; ok {
		t.Fatal("expected pending write to clear after final content arrives")
	}
}

func TestHarnessRegistryFileLifecycleAndHelpers(t *testing.T) {
	rootDir := t.TempDir()
	registry, err := newHarnessRegistry(rootDir, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		t.Fatalf("newHarnessRegistry() error = %v", err)
	}
	defer func() {
		if err := registry.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	relativePath := ".openase/harnesses/project/coding.md"
	if registry.Exists(relativePath) {
		t.Fatalf("Exists(%q) before Write should be false", relativePath)
	}

	content := "---\nworkflow:\n  role: coding\n---\n\n# Coding\n"
	if err := registry.Write(relativePath, content); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if !registry.Exists(relativePath) {
		t.Fatalf("Exists(%q) after Write should be true", relativePath)
	}

	got, err := registry.Read(relativePath)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got != content {
		t.Fatalf("Read() = %q, want %q", got, content)
	}

	absolutePath := registry.absolutePath(relativePath)
	roundTripPath, ok := registry.relativePath(absolutePath)
	if !ok || roundTripPath != relativePath {
		t.Fatalf("relativePath() = %q, %t", roundTripPath, ok)
	}
	if _, ok := registry.relativePath(filepath.Join(t.TempDir(), "outside.md")); ok {
		t.Fatal("relativePath(outside root) expected false")
	}

	if err := registry.Delete(relativePath); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if registry.Exists(relativePath) {
		t.Fatalf("Exists(%q) after Delete should be false", relativePath)
	}
	if err := registry.Delete(relativePath); err != nil {
		t.Fatalf("Delete() missing file error = %v", err)
	}
}

func TestHarnessRegistryCloseAllowsNilReceivers(t *testing.T) {
	if err := (*harnessRegistry)(nil).Close(); err != nil {
		t.Fatalf("(*harnessRegistry)(nil).Close() error = %v", err)
	}
	if err := (&harnessRegistry{}).Close(); err != nil {
		t.Fatalf("empty harnessRegistry.Close() error = %v", err)
	}
}

func TestHarnessRegistryReportsMissingReadsAndRejectsFileRoot(t *testing.T) {
	t.Parallel()

	rootFile := filepath.Join(t.TempDir(), "root-file")
	if err := os.WriteFile(rootFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write root file: %v", err)
	}
	if _, err := newHarnessRegistry(rootFile, slog.New(slog.NewTextHandler(io.Discard, nil)), nil); err == nil {
		t.Fatal("newHarnessRegistry(file root) expected error")
	}

	registry := &harnessRegistry{
		rootDir:    t.TempDir(),
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		cache:      map[string]cachedHarness{},
		pending:    map[string]string{},
		watchedDir: map[string]struct{}{},
	}
	if _, err := registry.Read(".openase/harnesses/missing.md"); err == nil {
		t.Fatal("Read(missing) expected error")
	}
}

func TestHarnessRegistryReloadsOnExternalChangesAndTracksNewDirs(t *testing.T) {
	rootDir := t.TempDir()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			t.Fatalf("watcher.Close() error = %v", err)
		}
	}()
	registry := &harnessRegistry{
		rootDir:    rootDir,
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		watcher:    watcher,
		cache:      map[string]cachedHarness{},
		pending:    map[string]string{},
		watchedDir: map[string]struct{}{},
		done:       make(chan struct{}),
	}
	if err := registry.addDirWatchRecursive(rootDir); err != nil {
		t.Fatalf("addDirWatchRecursive() error = %v", err)
	}

	relativePath := ".openase/harnesses/project/coding.md"
	initialContent := "---\nworkflow:\n  role: coding\n---\nbody\n"
	if err := registry.Write(relativePath, initialContent); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	reloadEvents := make([]harnessReloadEvent, 0, 1)
	registry.onReload = func(event harnessReloadEvent) {
		reloadEvents = append(reloadEvents, event)
	}

	absolutePath := registry.absolutePath(relativePath)
	registry.handleEvent(fsnotify.Event{Name: absolutePath, Op: fsnotify.Write})

	updatedContent := "---\nworkflow:\n  role: coding\n---\nupdated\n"
	if err := os.WriteFile(absolutePath, []byte(updatedContent), 0o600); err != nil {
		t.Fatalf("write updated harness: %v", err)
	}
	registry.handleEvent(fsnotify.Event{Name: absolutePath, Op: fsnotify.Write})

	if len(reloadEvents) != 1 {
		t.Fatalf("reloadEvents = %+v, want 1 event", reloadEvents)
	}
	if reloadEvents[0].PreviousContent != initialContent || reloadEvents[0].CurrentContent != updatedContent {
		t.Fatalf("reload event = %+v", reloadEvents[0])
	}

	newDir := filepath.Join(rootDir, "nested")
	if err := os.MkdirAll(newDir, 0o750); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	registry.handleEvent(fsnotify.Event{Name: newDir, Op: fsnotify.Create})
	if _, ok := registry.watchedDir[newDir]; !ok {
		t.Fatalf("expected new directory %q to be watched", newDir)
	}

	if err := os.Remove(absolutePath); err != nil {
		t.Fatalf("remove harness file: %v", err)
	}
	registry.handleEvent(fsnotify.Event{Name: absolutePath, Op: fsnotify.Remove})
	if registry.Exists(relativePath) {
		t.Fatalf("Exists(%q) after remove event should be false", relativePath)
	}
}

func TestHarnessRegistryWriteAndDeleteFailurePaths(t *testing.T) {
	rootDir := t.TempDir()
	registry, err := newHarnessRegistry(rootDir, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		t.Fatalf("newHarnessRegistry() error = %v", err)
	}
	defer func() {
		if err := registry.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	writePath := ".openase/harnesses/project/write-failure.md"
	writeAbsPath := registry.absolutePath(writePath)
	if err := os.MkdirAll(writeAbsPath, 0o755); err != nil {
		t.Fatalf("mkdir write failure path: %v", err)
	}
	if err := registry.Write(writePath, "---\nworkflow:\n  role: coding\n---\nbody\n"); err == nil {
		t.Fatal("Write(directory path) expected error")
	}
	if _, ok := registry.pending[writeAbsPath]; ok {
		t.Fatal("Write(directory path) should clear pending entry on failure")
	}
	if _, ok := registry.cache[writePath]; ok {
		t.Fatal("Write(directory path) should clear cache entry on failure")
	}

	deletePath := ".openase/harnesses/project/delete-failure.md"
	deleteAbsPath := registry.absolutePath(deletePath)
	if err := os.MkdirAll(deleteAbsPath, 0o755); err != nil {
		t.Fatalf("mkdir delete failure path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deleteAbsPath, "nested.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	if err := registry.Delete(deletePath); err == nil {
		t.Fatal("Delete(non-empty directory) expected error")
	}
}
