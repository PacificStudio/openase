package machinetransport

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/logging"
)

var _ = logging.DeclareComponent("machine-transport-sync")

func syncLocalArtifacts(request SyncArtifactsRequest) error {
	localRoot := filepath.Clean(strings.TrimSpace(request.LocalRoot))
	targetRoot := filepath.Clean(strings.TrimSpace(request.TargetRoot))
	if localRoot == "." || strings.TrimSpace(request.LocalRoot) == "" {
		return fmt.Errorf("local artifact root must not be empty")
	}
	if targetRoot == "." || strings.TrimSpace(request.TargetRoot) == "" {
		return fmt.Errorf("target artifact root must not be empty")
	}

	for _, relative := range request.RemovePaths {
		trimmed := strings.TrimSpace(relative)
		if trimmed == "" {
			continue
		}
		if err := removeLocalPath(filepath.Join(targetRoot, filepath.FromSlash(trimmed))); err != nil {
			return err
		}
	}
	for _, relative := range request.Paths {
		trimmed := strings.TrimSpace(relative)
		if trimmed == "" {
			continue
		}
		if err := copyLocalArtifact(localRoot, targetRoot, trimmed); err != nil {
			return err
		}
	}
	return nil
}

func copyLocalArtifact(localRoot string, targetRoot string, relative string) error {
	sourcePath := filepath.Join(localRoot, filepath.FromSlash(relative))
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat local artifact %s: %w", relative, err)
	}
	targetPath := filepath.Join(targetRoot, filepath.FromSlash(relative))
	if info.IsDir() {
		return filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relativePath, err := filepath.Rel(sourcePath, path)
			if err != nil {
				return fmt.Errorf("derive artifact relative path: %w", err)
			}
			destination := targetPath
			if relativePath != "." {
				destination = filepath.Join(targetPath, relativePath)
			}
			if entry.IsDir() {
				if err := os.MkdirAll(destination, 0o750); err != nil {
					return fmt.Errorf("create artifact directory %s: %w", destination, err)
				}
				return nil
			}
			return copyLocalFile(path, destination)
		})
	}
	return copyLocalFile(sourcePath, targetPath)
}

func copyLocalFile(sourcePath string, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
		return fmt.Errorf("create artifact parent %s: %w", filepath.Dir(targetPath), err)
	}
	// #nosec G304 -- artifact paths are derived from orchestrator-managed sync inputs.
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open artifact source %s: %w", sourcePath, err)
	}
	defer func() { _ = source.Close() }()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("stat artifact source %s: %w", sourcePath, err)
	}
	// #nosec G304 -- artifact paths are derived from orchestrator-managed sync inputs.
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open artifact target %s: %w", targetPath, err)
	}
	defer func() { _ = target.Close() }()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("copy artifact %s: %w", sourcePath, err)
	}
	return nil
}

func syncArtifactsWithSession(session CommandSession, request SyncArtifactsRequest) error {
	if session == nil {
		return fmt.Errorf("artifact sync session unavailable")
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("open stdin for artifact sync: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("open stderr for artifact sync: %w", err)
	}

	var stderrBuffer bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stderrBuffer, stderr)
		close(stderrDone)
	}()

	command := buildRemoteSyncCommand(request)
	if err := session.Start(command); err != nil {
		_ = stdin.Close()
		<-stderrDone
		return fmt.Errorf("start artifact sync: %w", err)
	}

	writer := tar.NewWriter(stdin)
	writeErr := writeArtifactArchive(writer, request.LocalRoot, request.Paths)
	closeErr := writer.Close()
	stdinCloseErr := stdin.Close()
	waitErr := session.Wait()
	<-stderrDone

	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if stdinCloseErr != nil {
		return stdinCloseErr
	}
	if waitErr != nil {
		return fmt.Errorf("%w: %s", waitErr, strings.TrimSpace(stderrBuffer.String()))
	}
	return nil
}

func buildRemoteSyncCommand(request SyncArtifactsRequest) string {
	lines := []string{
		"set -eu",
		"mkdir -p " + sshinfra.ShellQuote(strings.TrimSpace(request.TargetRoot)),
	}
	for _, relative := range request.RemovePaths {
		trimmed := strings.TrimSpace(relative)
		if trimmed == "" {
			continue
		}
		lines = append(lines, "rm -rf "+sshinfra.ShellQuote(filepath.ToSlash(filepath.Join(request.TargetRoot, filepath.FromSlash(trimmed)))))
	}
	lines = append(lines, "tar -C "+sshinfra.ShellQuote(strings.TrimSpace(request.TargetRoot))+" -xf -")
	return strings.Join(lines, " && ")
}

func writeArtifactArchive(writer *tar.Writer, localRoot string, paths []string) error {
	root := filepath.Clean(strings.TrimSpace(localRoot))
	if root == "." || strings.TrimSpace(localRoot) == "" {
		return fmt.Errorf("local artifact root must not be empty")
	}
	for _, relative := range paths {
		trimmed := strings.TrimSpace(relative)
		if trimmed == "" {
			continue
		}
		sourcePath := filepath.Join(root, filepath.FromSlash(trimmed))
		info, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("stat artifact %s: %w", trimmed, err)
		}
		if info.IsDir() {
			if err := filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				info, err := entry.Info()
				if err != nil {
					return err
				}
				relativePath, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				return writeArtifactHeader(writer, path, filepath.ToSlash(relativePath), info)
			}); err != nil {
				return fmt.Errorf("archive artifact %s: %w", trimmed, err)
			}
			continue
		}
		if err := writeArtifactHeader(writer, sourcePath, filepath.ToSlash(trimmed), info); err != nil {
			return fmt.Errorf("archive artifact %s: %w", trimmed, err)
		}
	}
	return nil
}

func writeArtifactHeader(writer *tar.Writer, sourcePath string, archivePath string, info os.FileInfo) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = archivePath
	if info.IsDir() {
		if !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		return nil
	}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	// #nosec G304 -- archived file paths are derived from validated sync inputs.
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.Copy(writer, file)
	return err
}
