package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
)

const (
	skillBundleFileKindEntrypoint = "entrypoint"
	skillBundleFileKindMetadata   = "metadata"
	skillBundleFileKindScript     = "script"
	skillBundleFileKindReference  = "reference"
	skillBundleFileKindAsset      = "asset"

	skillBundleEncodingUTF8   = "utf8"
	skillBundleEncodingBinary = "binary"
)

type SkillBundleFileInput = domain.SkillBundleFileInput

type SkillBundleFile = domain.SkillBundleFile

type SkillBundle = domain.SkillBundle

func buildSingleFileSkillBundle(name string, content string, fallbackDescription string) (SkillBundle, error) {
	normalizedContent, err := ensureSkillContent(name, content, fallbackDescription)
	if err != nil {
		return SkillBundle{}, err
	}
	return parseSkillBundle(name, []SkillBundleFileInput{
		{
			Path:      "SKILL.md",
			Content:   []byte(normalizedContent),
			MediaType: "text/markdown; charset=utf-8",
		},
	})
}

func parseSkillBundle(name string, files []SkillBundleFileInput) (SkillBundle, error) {
	normalizedNames, err := normalizeSkillNames([]string{name})
	if err != nil {
		return SkillBundle{}, err
	}
	skillName := normalizedNames[0]
	if len(files) == 0 {
		return SkillBundle{}, fmt.Errorf("%w: skill bundle must contain at least SKILL.md", ErrSkillInvalid)
	}

	normalized := make([]SkillBundleFile, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	var entrypoint *SkillBundleFile
	var totalSize int64
	for _, item := range files {
		cleanPath, err := normalizeSkillBundlePath(item.Path)
		if err != nil {
			return SkillBundle{}, err
		}
		if _, ok := seen[cleanPath]; ok {
			return SkillBundle{}, fmt.Errorf("%w: duplicate skill bundle path %q", ErrSkillInvalid, cleanPath)
		}
		seen[cleanPath] = struct{}{}

		file := SkillBundleFile{
			Path:         cleanPath,
			FileKind:     inferSkillFileKind(cleanPath),
			MediaType:    normalizeSkillMediaType(item.MediaType, item.Content),
			Encoding:     inferSkillEncoding(item.Content),
			IsExecutable: item.IsExecutable,
			SizeBytes:    int64(len(item.Content)),
			SHA256:       contentHashBytes(item.Content),
			Content:      append([]byte(nil), item.Content...),
		}
		normalized = append(normalized, file)
		totalSize += file.SizeBytes
		if cleanPath == "SKILL.md" {
			entrypoint = &normalized[len(normalized)-1]
		}
	}

	if entrypoint == nil {
		return SkillBundle{}, fmt.Errorf("%w: skill bundle must contain SKILL.md", ErrSkillInvalid)
	}
	if !utf8.Valid(entrypoint.Content) {
		return SkillBundle{}, fmt.Errorf("%w: SKILL.md must be valid UTF-8", ErrSkillInvalid)
	}

	document, body, err := parseSkillDocument(string(entrypoint.Content))
	if err != nil {
		return SkillBundle{}, fmt.Errorf("%w: %s", ErrSkillInvalid, err)
	}
	if document.Name != skillName {
		return SkillBundle{}, fmt.Errorf("%w: skill frontmatter name %q must match %q", ErrSkillInvalid, document.Name, skillName)
	}

	sort.Slice(normalized, func(i int, j int) bool {
		return normalized[i].Path < normalized[j].Path
	})
	bundleHash := hashSkillBundleFiles(normalized)
	manifest := buildSkillBundleManifest(skillName, document.Description, normalized, bundleHash, entrypoint.Path, totalSize)

	return SkillBundle{
		Name:             skillName,
		Description:      skillDescriptionFromSkillDocument(document, body),
		Files:            normalized,
		FileCount:        len(normalized),
		SizeBytes:        totalSize,
		BundleHash:       bundleHash,
		Manifest:         manifest,
		EntrypointPath:   entrypoint.Path,
		EntrypointSHA256: entrypoint.SHA256,
		EntrypointBody:   string(entrypoint.Content),
	}, nil
}

func normalizeSkillBundlePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: skill bundle file path must not be empty", ErrSkillInvalid)
	}
	if strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("%w: skill bundle path %q must use forward slashes", ErrSkillInvalid, raw)
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", fmt.Errorf("%w: skill bundle path %q must be relative", ErrSkillInvalid, raw)
	}
	clean := path.Clean(trimmed)
	switch {
	case clean == ".", clean == "..", strings.HasPrefix(clean, "../"):
		return "", fmt.Errorf("%w: skill bundle path %q escapes bundle root", ErrSkillInvalid, raw)
	case strings.HasPrefix(clean, "./"):
		return "", fmt.Errorf("%w: skill bundle path %q must be normalized", ErrSkillInvalid, raw)
	}
	return clean, nil
}

func inferSkillFileKind(filePath string) string {
	switch {
	case filePath == "SKILL.md":
		return skillBundleFileKindEntrypoint
	case filePath == "agents/openai.yaml":
		return skillBundleFileKindMetadata
	case strings.HasPrefix(filePath, "scripts/"):
		return skillBundleFileKindScript
	case strings.HasPrefix(filePath, "references/"):
		return skillBundleFileKindReference
	default:
		return skillBundleFileKindAsset
	}
}

func inferSkillEncoding(content []byte) string {
	if utf8.Valid(content) {
		return skillBundleEncodingUTF8
	}
	return skillBundleEncodingBinary
}

func normalizeSkillMediaType(raw string, content []byte) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		return trimmed
	}
	if len(content) == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(content)
}

func hashSkillBundleFiles(files []SkillBundleFile) string {
	sum := sha256.New()
	for _, file := range files {
		sum.Write([]byte(file.Path))
		sum.Write([]byte{0})
		sum.Write([]byte(file.FileKind))
		sum.Write([]byte{0})
		if file.IsExecutable {
			sum.Write([]byte{1})
		} else {
			sum.Write([]byte{0})
		}
		sum.Write([]byte{0})
		sum.Write([]byte(file.SHA256))
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func contentHashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func buildSkillBundleManifest(
	name string,
	description string,
	files []SkillBundleFile,
	bundleHash string,
	entrypointPath string,
	totalSize int64,
) map[string]any {
	fileItems := make([]map[string]any, 0, len(files))
	for _, file := range files {
		fileItems = append(fileItems, map[string]any{
			"path":          file.Path,
			"file_kind":     file.FileKind,
			"media_type":    file.MediaType,
			"encoding":      file.Encoding,
			"is_executable": file.IsExecutable,
			"size_bytes":    file.SizeBytes,
			"sha256":        file.SHA256,
		})
	}
	return map[string]any{
		"name":            name,
		"description":     description,
		"entrypoint_path": entrypointPath,
		"bundle_hash":     bundleHash,
		"file_count":      len(files),
		"size_bytes":      totalSize,
		"files":           fileItems,
	}
}

func skillDescriptionFromSkillDocument(document skillDocument, body string) string {
	title := parseSkillTitle(body)
	if strings.TrimSpace(title) != "" {
		return title
	}
	return strings.TrimSpace(document.Description)
}
