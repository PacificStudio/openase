package chat

import (
	"testing"
)

func TestParseWorkspaceCreatableFilePathRejectsTraversal(t *testing.T) {
	t.Parallel()

	if _, err := ParseWorkspaceCreatableFilePath("../secrets.txt"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestNormalizeWorkspaceTextContentUsesRequestedLineEnding(t *testing.T) {
	t.Parallel()

	content := normalizeWorkspaceTextContent(WorkspaceTextContent("a\r\nb\n"), WorkspaceLineEndingCRLF)
	if string(content) != "a\r\nb\r\n" {
		t.Fatalf("content = %q, want CRLF normalized text", string(content))
	}
}
