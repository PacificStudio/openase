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
