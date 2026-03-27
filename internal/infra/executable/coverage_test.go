package executable

import "testing"

func TestPathResolverLookPath(t *testing.T) {
	t.Parallel()

	resolver := NewPathResolver()
	if resolver == nil {
		t.Fatal("NewPathResolver() = nil")
	}
	if path, err := resolver.LookPath("sh"); err != nil || path == "" {
		t.Fatalf("LookPath(sh) = %q, %v", path, err)
	}
	if _, err := resolver.LookPath("openase-command-that-does-not-exist"); err == nil {
		t.Fatal("LookPath(missing) expected error")
	}
}
