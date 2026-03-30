package projectrepomirror

import (
	"path/filepath"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
)

func TestDeriveMirrorRootUsesLocalDefault(t *testing.T) {
	t.Setenv("HOME", filepath.Join(string(filepath.Separator), "tmp", "openase-home"))

	root, err := deriveMirrorRoot(&ent.Machine{
		Name: "local",
		Host: "local",
	})
	if err != nil {
		t.Fatalf("deriveMirrorRoot() error = %v", err)
	}

	expected := filepath.Join(string(filepath.Separator), "tmp", "openase-home", ".openase", "mirrors")
	if root != expected {
		t.Fatalf("deriveMirrorRoot() = %q, want %q", root, expected)
	}
}
