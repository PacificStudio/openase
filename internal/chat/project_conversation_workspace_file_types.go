package chat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type WorkspaceRepoPath string

func (p WorkspaceRepoPath) String() string { return string(p) }

type WorkspaceFilePath string

func (p WorkspaceFilePath) String() string { return string(p) }

type WorkspaceCreatableFilePath string

func (p WorkspaceCreatableFilePath) String() string { return string(p) }

type WorkspaceRenamableFilePath string

func (p WorkspaceRenamableFilePath) String() string { return string(p) }

type WorkspaceDeleteableFilePath string

func (p WorkspaceDeleteableFilePath) String() string { return string(p) }

type WorkspaceFileRevision string

func (r WorkspaceFileRevision) String() string { return string(r) }

type WorkspaceEncoding string

const WorkspaceEncodingUTF8 WorkspaceEncoding = "utf-8"

func (e WorkspaceEncoding) String() string { return string(e) }

type WorkspaceLineEnding string

const (
	WorkspaceLineEndingLF   WorkspaceLineEnding = "lf"
	WorkspaceLineEndingCRLF WorkspaceLineEnding = "crlf"
)

func (l WorkspaceLineEnding) String() string { return string(l) }

type WorkspaceTextContent string

func (c WorkspaceTextContent) String() string { return string(c) }

type ProjectConversationWorkspaceReadOnlyReason string

const (
	ProjectConversationWorkspaceReadOnlyReasonBinaryFile          ProjectConversationWorkspaceReadOnlyReason = "binary_file"
	ProjectConversationWorkspaceReadOnlyReasonFileTooLarge        ProjectConversationWorkspaceReadOnlyReason = "file_too_large"
	ProjectConversationWorkspaceReadOnlyReasonUnsupportedEncoding ProjectConversationWorkspaceReadOnlyReason = "unsupported_encoding"
)

func (r ProjectConversationWorkspaceReadOnlyReason) String() string { return string(r) }

type ProjectConversationWorkspaceFileSaveInput struct {
	RepoPath     WorkspaceRepoPath
	Path         WorkspaceFilePath
	BaseRevision WorkspaceFileRevision
	Content      WorkspaceTextContent
	Encoding     WorkspaceEncoding
	LineEnding   WorkspaceLineEnding
}

type ProjectConversationWorkspaceFileDraftContext struct {
	RepoPath   WorkspaceRepoPath
	Path       WorkspaceFilePath
	Content    WorkspaceTextContent
	Encoding   WorkspaceEncoding
	LineEnding WorkspaceLineEnding
}

type ProjectConversationWorkspaceFileCreateInput struct {
	RepoPath WorkspaceRepoPath
	Path     WorkspaceCreatableFilePath
}

type ProjectConversationWorkspaceFileRenameInput struct {
	RepoPath WorkspaceRepoPath
	FromPath WorkspaceRenamableFilePath
	ToPath   WorkspaceCreatableFilePath
}

type ProjectConversationWorkspaceFileDeleteInput struct {
	RepoPath WorkspaceRepoPath
	Path     WorkspaceDeleteableFilePath
}

type ProjectConversationWorkspaceFileSaved struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
	Revision       string
	SizeBytes      int64
	Encoding       string
	LineEnding     string
}

type ProjectConversationWorkspaceFileCreated struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
	Revision       string
	SizeBytes      int64
	Encoding       string
	LineEnding     string
}

type ProjectConversationWorkspaceFileRenamed struct {
	ConversationID uuid.UUID
	RepoPath       string
	FromPath       string
	ToPath         string
}

type ProjectConversationWorkspaceFileDeleted struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
}

type ProjectConversationWorkspaceFileConflictError struct {
	CurrentFile ProjectConversationWorkspaceFilePreview
}

func (e *ProjectConversationWorkspaceFileConflictError) Error() string {
	if e == nil {
		return "project conversation workspace file conflict"
	}
	return fmt.Sprintf(
		"project conversation workspace file revision conflict for %s",
		strings.TrimSpace(e.CurrentFile.Path),
	)
}

type ProjectConversationWorkspaceFileReadOnlyError struct {
	Reason ProjectConversationWorkspaceReadOnlyReason
}

func (e *ProjectConversationWorkspaceFileReadOnlyError) Error() string {
	if e == nil || strings.TrimSpace(e.Reason.String()) == "" {
		return "project conversation workspace file is read-only"
	}
	return fmt.Sprintf("project conversation workspace file is read-only: %s", e.Reason)
}

func ParseWorkspaceRepoPath(raw string) (WorkspaceRepoPath, error) {
	parsed, err := parseProjectConversationWorkspaceRelativePath(raw, false)
	if err != nil {
		return "", err
	}
	return WorkspaceRepoPath(parsed), nil
}

func ParseWorkspaceFilePath(raw string) (WorkspaceFilePath, error) {
	parsed, err := parseProjectConversationWorkspaceRelativePath(raw, false)
	if err != nil {
		return "", err
	}
	return WorkspaceFilePath(parsed), nil
}

func ParseWorkspaceCreatableFilePath(raw string) (WorkspaceCreatableFilePath, error) {
	parsed, err := parseProjectConversationWorkspaceRelativePath(raw, false)
	if err != nil {
		return "", err
	}
	return WorkspaceCreatableFilePath(parsed), nil
}

func ParseWorkspaceRenamableFilePath(raw string) (WorkspaceRenamableFilePath, error) {
	parsed, err := parseProjectConversationWorkspaceRelativePath(raw, false)
	if err != nil {
		return "", err
	}
	return WorkspaceRenamableFilePath(parsed), nil
}

func ParseWorkspaceDeleteableFilePath(raw string) (WorkspaceDeleteableFilePath, error) {
	parsed, err := parseProjectConversationWorkspaceRelativePath(raw, false)
	if err != nil {
		return "", err
	}
	return WorkspaceDeleteableFilePath(parsed), nil
}

func ParseWorkspaceFileRevision(raw string) (WorkspaceFileRevision, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("base_revision must not be empty")
	}
	return WorkspaceFileRevision(trimmed), nil
}

func ParseWorkspaceEncoding(raw string) (WorkspaceEncoding, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(WorkspaceEncodingUTF8):
		return WorkspaceEncodingUTF8, nil
	default:
		return "", fmt.Errorf("encoding must be utf-8")
	}
}

func ParseWorkspaceLineEnding(raw string) (WorkspaceLineEnding, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(WorkspaceLineEndingLF):
		return WorkspaceLineEndingLF, nil
	case string(WorkspaceLineEndingCRLF):
		return WorkspaceLineEndingCRLF, nil
	default:
		return "", fmt.Errorf("line_ending must be lf or crlf")
	}
}

func ParseWorkspaceTextContent(raw string) (WorkspaceTextContent, error) {
	return WorkspaceTextContent(raw), nil
}

func normalizeWorkspaceTextContent(
	content WorkspaceTextContent,
	lineEnding WorkspaceLineEnding,
) []byte {
	normalized := strings.ReplaceAll(content.String(), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if lineEnding == WorkspaceLineEndingCRLF {
		normalized = strings.ReplaceAll(normalized, "\n", "\r\n")
	}
	return []byte(normalized)
}

func detectWorkspaceLineEnding(content []byte) WorkspaceLineEnding {
	if bytes.Contains(content, []byte("\r\n")) {
		return WorkspaceLineEndingCRLF
	}
	return WorkspaceLineEndingLF
}

func computeWorkspaceFileRevision(content []byte) WorkspaceFileRevision {
	sum := sha256.Sum256(content)
	return WorkspaceFileRevision(hex.EncodeToString(sum[:]))
}

func classifyWorkspaceTextFile(content []byte, sizeBytes int64) (
	previewKind ProjectConversationWorkspacePreviewKind,
	writable bool,
	reason ProjectConversationWorkspaceReadOnlyReason,
	lineEnding WorkspaceLineEnding,
) {
	lineEnding = detectWorkspaceLineEnding(content)
	if projectConversationWorkspaceLooksBinary(content) {
		return ProjectConversationWorkspacePreviewKindBinary, false, ProjectConversationWorkspaceReadOnlyReasonBinaryFile, lineEnding
	}
	if sizeBytes > projectConversationWorkspacePreviewLimitBytes {
		return ProjectConversationWorkspacePreviewKindText, false, ProjectConversationWorkspaceReadOnlyReasonFileTooLarge, lineEnding
	}
	return ProjectConversationWorkspacePreviewKindText, true, "", lineEnding
}
