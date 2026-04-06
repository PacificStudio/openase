package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePathRemovesEmptyAndDuplicateEntries(t *testing.T) {
	raw := strings.Join([]string{"/usr/bin", "", "/usr/local/bin", "/usr/bin", " /opt/bin "}, string(os.PathListSeparator))
	got := NormalizePath(raw)
	want := strings.Join([]string{"/usr/bin", "/usr/local/bin", "/opt/bin"}, string(os.PathListSeparator))
	if got != want {
		t.Fatalf("NormalizePath() = %q, want %q", got, want)
	}
}

func TestUpsertCreatesAndMergesAssignments(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := Upsert(path, map[string]string{
		"OPENASE_AUTH_TOKEN": "token-1",
		"PATH":               "/usr/bin:/opt/bin",
	}); err != nil {
		t.Fatalf("Upsert(create) error = %v", err)
	}
	if err := Upsert(path, map[string]string{
		"PATH": "/usr/bin:/custom/bin",
	}); err != nil {
		t.Fatalf("Upsert(update) error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)
	want := "OPENASE_AUTH_TOKEN=token-1\nPATH=/usr/bin:/custom/bin\n"
	if got != want {
		t.Fatalf("env content = %q, want %q", got, want)
	}
}

func TestUpsertPreservesCommentsAndUnrelatedAssignments(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	initial := "# comment\nOPENASE_AUTH_TOKEN=old\nCUSTOM=true\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := Upsert(path, map[string]string{
		"OPENASE_AUTH_TOKEN": "new",
		"PATH":               "/usr/bin",
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)
	want := "# comment\nOPENASE_AUTH_TOKEN=new\nCUSTOM=true\nPATH=/usr/bin\n"
	if got != want {
		t.Fatalf("env content = %q, want %q", got, want)
	}
}
