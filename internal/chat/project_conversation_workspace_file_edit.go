package chat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type WorkspaceRepoPath string

func (p WorkspaceRepoPath) String() string { return string(p) }

type WorkspaceFilePath string

func (p WorkspaceFilePath) String() string { return string(p) }

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

type ProjectConversationWorkspaceFileSaved struct {
	ConversationID uuid.UUID
	RepoPath       string
	Path           string
	Revision       string
	SizeBytes      int64
	Encoding       string
	LineEnding     string
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

func (s *ProjectConversationService) SaveWorkspaceFile(
	ctx context.Context,
	userID UserID,
	conversationID uuid.UUID,
	input ProjectConversationWorkspaceFileSaveInput,
) (ProjectConversationWorkspaceFileSaved, error) {
	resolved, relativePath, err := s.resolveConversationWorkspaceRepoPath(
		ctx,
		userID,
		conversationID,
		input.RepoPath.String(),
		input.Path.String(),
		false,
	)
	if err != nil {
		var conflictErr *ProjectConversationWorkspaceFileConflictError
		if errors.As(err, &conflictErr) && conflictErr != nil {
			conflictErr.CurrentFile.ConversationID = resolved.conversationID
			conflictErr.CurrentFile.RepoPath = resolved.repo.relativePath
			conflictErr.CurrentFile.Path = relativePath
		}
		return ProjectConversationWorkspaceFileSaved{}, err
	}

	content := normalizeWorkspaceTextContent(input.Content, input.LineEnding)
	revision, sizeBytes, err := s.writeConversationWorkspaceFile(
		ctx,
		resolved.machine,
		resolved.repo.repoPath,
		relativePath,
		input.BaseRevision,
		content,
		input.Encoding,
		input.LineEnding,
	)
	if err != nil {
		return ProjectConversationWorkspaceFileSaved{}, err
	}

	return ProjectConversationWorkspaceFileSaved{
		ConversationID: resolved.conversationID,
		RepoPath:       resolved.repo.relativePath,
		Path:           input.Path.String(),
		Revision:       revision.String(),
		SizeBytes:      sizeBytes,
		Encoding:       input.Encoding.String(),
		LineEnding:     input.LineEnding.String(),
	}, nil
}

func (s *ProjectConversationService) writeConversationWorkspaceFile(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
	baseRevision WorkspaceFileRevision,
	content []byte,
	encoding WorkspaceEncoding,
	lineEnding WorkspaceLineEnding,
) (WorkspaceFileRevision, int64, error) {
	if machine.Host != catalogdomain.LocalMachineHost {
		return s.writeRemoteConversationWorkspaceFile(
			ctx,
			machine,
			repoRoot,
			relativePath,
			baseRevision,
			content,
			encoding,
			lineEnding,
		)
	}
	return writeLocalConversationWorkspaceFile(repoRoot, relativePath, baseRevision, content, encoding, lineEnding)
}

func writeLocalConversationWorkspaceFile(
	repoRoot string,
	relativePath string,
	baseRevision WorkspaceFileRevision,
	content []byte,
	encoding WorkspaceEncoding,
	_ WorkspaceLineEnding,
) (WorkspaceFileRevision, int64, error) {
	if encoding != WorkspaceEncodingUTF8 {
		return "", 0, fmt.Errorf("workspace encoding %s is unsupported", encoding)
	}
	resolvedFile, err := resolveLocalProjectConversationWorkspaceFile(repoRoot, relativePath)
	if err != nil {
		return "", 0, err
	}

	currentContent, info, err := readLocalWorkspaceFileContent(resolvedFile.path)
	if err != nil {
		return "", 0, err
	}
	currentRevision := computeWorkspaceFileRevision(currentContent)
	if currentRevision != baseRevision {
		return "", 0, &ProjectConversationWorkspaceFileConflictError{
			CurrentFile: buildWorkspacePreviewFromContent(relativePath, currentContent, info.Size()),
		}
	}

	preview := buildWorkspacePreviewFromContent(relativePath, currentContent, info.Size())
	if !preview.Writable {
		return "", 0, &ProjectConversationWorkspaceFileReadOnlyError{Reason: ProjectConversationWorkspaceReadOnlyReason(preview.ReadOnlyReason)}
	}

	if err := writeFileAtomically(resolvedFile.path, content, info.Mode().Perm()); err != nil {
		return "", 0, err
	}

	return computeWorkspaceFileRevision(content), int64(len(content)), nil
}

func readLocalWorkspaceFileContent(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, ErrProjectConversationWorkspaceEntryNotFound
		}
		return nil, nil, fmt.Errorf("stat workspace file %s: %w", path, err)
	}
	// #nosec G304 -- path is resolved through resolveLocalProjectConversationWorkspaceFile and remains within the repo root.
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, ErrProjectConversationWorkspaceEntryNotFound
		}
		return nil, nil, fmt.Errorf("read workspace file %s: %w", path, err)
	}
	return content, info, nil
}

func buildWorkspacePreviewFromContent(
	relativePath string,
	content []byte,
	sizeBytes int64,
) ProjectConversationWorkspaceFilePreview {
	snippet := content
	truncated := int64(len(snippet)) > projectConversationWorkspacePreviewLimitBytes || sizeBytes > projectConversationWorkspacePreviewLimitBytes
	if int64(len(snippet)) > projectConversationWorkspacePreviewLimitBytes {
		snippet = snippet[:projectConversationWorkspacePreviewLimitBytes]
	}
	previewKind, writable, reason, lineEnding := classifyWorkspaceTextFile(content, sizeBytes)
	preview := ProjectConversationWorkspaceFilePreview{
		SizeBytes:      sizeBytes,
		MediaType:      projectConversationWorkspaceMediaType(relativePath),
		PreviewKind:    previewKind,
		Truncated:      truncated,
		Revision:       computeWorkspaceFileRevision(content).String(),
		Writable:       writable,
		ReadOnlyReason: reason.String(),
		Encoding:       WorkspaceEncodingUTF8.String(),
		LineEnding:     lineEnding.String(),
	}
	if previewKind == ProjectConversationWorkspacePreviewKindText {
		preview.Content = string(snippet)
	}
	return preview
}

func writeFileAtomically(path string, content []byte, mode os.FileMode) error {
	dirPath := filepath.Dir(path)
	base := filepath.Base(path)
	tempFile, err := os.CreateTemp(dirPath, "."+base+".openase-save-*")
	if err != nil {
		return fmt.Errorf("create temp workspace file for %s: %w", path, err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err := tempFile.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp workspace file %s: %w", tempPath, err)
	}
	if _, err := tempFile.Write(content); err != nil {
		return fmt.Errorf("write temp workspace file %s: %w", tempPath, err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync temp workspace file %s: %w", tempPath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp workspace file %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp workspace file %s into %s: %w", tempPath, path, err)
	}

	// #nosec G304 -- dirPath is derived from the validated resolved workspace file path above.
	dir, err := os.Open(dirPath)
	if err == nil {
		defer func() { _ = dir.Close() }()
		if syncErr := dir.Sync(); syncErr != nil {
			return fmt.Errorf("sync workspace directory %s: %w", dirPath, syncErr)
		}
	}
	return nil
}

func (s *ProjectConversationService) writeRemoteConversationWorkspaceFile(
	ctx context.Context,
	machine catalogdomain.Machine,
	repoRoot string,
	relativePath string,
	baseRevision WorkspaceFileRevision,
	content []byte,
	encoding WorkspaceEncoding,
	_ WorkspaceLineEnding,
) (WorkspaceFileRevision, int64, error) {
	if encoding != WorkspaceEncodingUTF8 {
		return "", 0, fmt.Errorf("workspace encoding %s is unsupported", encoding)
	}
	if s == nil || s.sshPool == nil {
		return "", 0, fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
	}
	client, err := s.sshPool.Get(ctx, machine)
	if err != nil {
		return "", 0, err
	}
	session, err := client.NewSession()
	if err != nil {
		return "", 0, fmt.Errorf("open ssh session for workspace file save: %w", err)
	}
	defer func() { _ = session.Close() }()

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", 0, fmt.Errorf("open ssh stdin for workspace file save: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return "", 0, fmt.Errorf("open ssh stdout for workspace file save: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return "", 0, fmt.Errorf("open ssh stderr for workspace file save: %w", err)
	}

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdoutBuffer, stdout)
		close(stdoutDone)
	}()
	go func() {
		_, _ = io.Copy(&stderrBuffer, stderr)
		close(stderrDone)
	}()

	command := buildRemoteWorkspaceFileSaveCommand(repoRoot, relativePath, baseRevision)
	if err := session.Start(command); err != nil {
		_ = stdin.Close()
		<-stdoutDone
		<-stderrDone
		return "", 0, fmt.Errorf("start ssh workspace file save: %w", err)
	}
	_, writeErr := stdin.Write(content)
	stdinCloseErr := stdin.Close()
	waitErr := session.Wait()
	<-stdoutDone
	<-stderrDone

	if writeErr != nil {
		return "", 0, fmt.Errorf("stream workspace file content for %s: %w", relativePath, writeErr)
	}
	if stdinCloseErr != nil {
		return "", 0, fmt.Errorf("close workspace file save stdin for %s: %w", relativePath, stdinCloseErr)
	}

	stdoutText := strings.TrimSpace(stdoutBuffer.String())
	stderrText := strings.TrimSpace(stderrBuffer.String())
	if waitErr != nil {
		if strings.HasPrefix(stdoutText, "conflict\t") {
			currentFile, parseErr := parseRemoteWorkspaceConflictPreview(stdoutText)
			if parseErr != nil {
				return "", 0, parseErr
			}
			return "", 0, &ProjectConversationWorkspaceFileConflictError{CurrentFile: currentFile}
		}
		if strings.Contains(stderrText, "readonly:") {
			reason := strings.TrimSpace(strings.TrimPrefix(stderrText, "readonly:"))
			return "", 0, &ProjectConversationWorkspaceFileReadOnlyError{Reason: ProjectConversationWorkspaceReadOnlyReason(reason)}
		}
		if strings.Contains(stderrText, "missing") {
			return "", 0, ErrProjectConversationWorkspaceEntryNotFound
		}
		if strings.Contains(stderrText, "escape") {
			return "", 0, ErrProjectConversationWorkspacePathInvalid
		}
		return "", 0, fmt.Errorf("%w: %s", waitErr, stderrText)
	}

	fields := strings.SplitN(stdoutText, "\t", 2)
	if len(fields) != 2 {
		return "", 0, fmt.Errorf("parse remote workspace file save output %q", stdoutText)
	}
	var size int64
	if _, err := fmt.Sscanf(fields[1], "%d", &size); err != nil {
		return "", 0, fmt.Errorf("parse remote workspace file save size %q: %w", fields[1], err)
	}
	return WorkspaceFileRevision(strings.TrimSpace(fields[0])), size, nil
}

func buildRemoteWorkspaceFileSaveCommand(
	repoRoot string,
	relativePath string,
	baseRevision WorkspaceFileRevision,
) string {
	return fmt.Sprintf(`sh -lc %s`, projectConversationShellQuote(fmt.Sprintf(`set -eu
repo=%s
relative=%s
expected_revision=%s
repo_real=$(cd "$repo" && pwd -P)
base="$relative"
parent=""
if [ "$relative" != "$base" ]; then
  parent="${relative%%/*}"
fi
target_dir="$repo"
if [ -n "$parent" ]; then
  target_dir="$repo/$parent"
fi
dir_real=$(cd "$target_dir" 2>/dev/null && pwd -P) || { echo missing >&2; exit 11; }
case "$dir_real" in
  "$repo_real"|"$repo_real"/*) ;;
  *) echo escape >&2; exit 12 ;;
esac
target="$dir_real/$base"
if [ -L "$target" ]; then
  echo escape >&2
  exit 12
fi
if [ ! -f "$target" ]; then
  echo missing >&2
  exit 11
fi
current_size=$(wc -c <"$target" | tr -d '[:space:]')
current_revision=$(sha256sum "$target" | awk '{print $1}')
head -c 262145 "$target" | base64 -w0 >"$target.dirpreview.openase" 2>/dev/null || true
preview_base64=$(cat "$target.dirpreview.openase" 2>/dev/null || printf '')
rm -f "$target.dirpreview.openase"
if [ "$current_revision" != "$expected_revision" ]; then
  printf 'conflict\t%%s\t%%s\t%%s\n' "$current_revision" "$current_size" "$preview_base64"
  exit 13
fi
if python3 - "$target" <<'PY' >/dev/null 2>&1
import pathlib, sys
path = pathlib.Path(sys.argv[1])
content = path.read_bytes()
if b'\x00' in content:
    raise SystemExit(1)
content.decode('utf-8')
if len(content) > 262144:
    raise SystemExit(2)
PY
then
  :
else
  status=$?
  if [ "$status" = 1 ]; then
    echo readonly:binary_file >&2
  elif [ "$status" = 2 ]; then
    echo readonly:file_too_large >&2
  else
    echo readonly:unsupported_encoding >&2
  fi
  exit 14
fi
mode=$(stat -c '%%a' "$target")
tmp=$(mktemp "$dir_real/.openase-save.XXXXXX")
trap 'rm -f "$tmp"' EXIT
cat >"$tmp"
chmod "$mode" "$tmp"
mv "$tmp" "$target"
new_size=$(wc -c <"$target" | tr -d '[:space:]')
new_revision=$(sha256sum "$target" | awk '{print $1}')
printf '%%s\t%%s' "$new_revision" "$new_size"
`, projectConversationShellQuote(repoRoot), projectConversationShellQuote(relativePath), projectConversationShellQuote(baseRevision.String()))))
}

func parseRemoteWorkspaceConflictPreview(output string) (ProjectConversationWorkspaceFilePreview, error) {
	fields := strings.SplitN(strings.TrimSpace(output), "\t", 4)
	if len(fields) != 4 || fields[0] != "conflict" {
		return ProjectConversationWorkspaceFilePreview{}, fmt.Errorf("parse remote workspace file conflict output %q", output)
	}
	var size int64
	if _, err := fmt.Sscanf(fields[2], "%d", &size); err != nil {
		return ProjectConversationWorkspaceFilePreview{}, fmt.Errorf("parse remote workspace file conflict size %q: %w", fields[2], err)
	}
	content, err := base64.StdEncoding.DecodeString(fields[3])
	if err != nil {
		return ProjectConversationWorkspaceFilePreview{}, fmt.Errorf("decode remote workspace file conflict preview: %w", err)
	}
	preview := buildWorkspacePreviewFromContent("", content, size)
	preview.Revision = strings.TrimSpace(fields[1])
	return preview, nil
}
