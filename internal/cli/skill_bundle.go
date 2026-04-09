package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type skillImportPayload struct {
	Name      string                   `json:"name"`
	IsEnabled *bool                    `json:"is_enabled,omitempty"`
	Files     []skillImportPayloadFile `json:"files"`
}

type skillImportPayloadFile struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
	MediaType     string `json:"media_type,omitempty"`
	IsExecutable  bool   `json:"is_executable,omitempty"`
}

func newSkillImportCommand() *cobra.Command {
	var apiOptions apiCommandOptions
	var output apiOutputOptions
	var nameOverride string
	var enableNow bool
	var disableNow bool

	spec := openAPICommandSpec{
		Method: http.MethodPost,
		Path:   "/api/v1/projects/{projectId}/skills/import",
	}
	command := &cobra.Command{
		Use:   "import [projectId] [dir]",
		Short: "Import a local skill bundle directory into a project.",
		Long: strings.TrimSpace(`
Import a local skill bundle directory into a project.

The CLI reads the local directory, packages regular files under that root, and
uploads the bundle to the OpenASE API. The server remains the authority for
bundle validation and persistence; the CLI never asks the server to read an
arbitrary filesystem path on its own host.

projectId must be a UUID value that identifies the target OpenASE project.
`),
		Example: strings.TrimSpace(`
openase skill import 11111111-1111-1111-1111-111111111111 ./skills/deploy-openase
openase skill import $OPENASE_PROJECT_ID ./skills/deploy-openase --enable
`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if enableNow && disableNow {
				return fmt.Errorf("only one of --enable or --disable can be set")
			}

			projectID := strings.TrimSpace(args[0])
			dir := strings.TrimSpace(args[1])
			if projectID == "" {
				return fmt.Errorf("projectId must not be empty")
			}
			if dir == "" {
				return fmt.Errorf("dir must not be empty")
			}

			payload, err := buildSkillImportPayload(dir, strings.TrimSpace(nameOverride), enableFlagValue(enableNow, disableNow))
			if err != nil {
				return err
			}
			body, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("marshal skill import payload: %w", err)
			}

			apiContext, err := apiOptions.resolve()
			if err != nil {
				return err
			}
			response, err := apiContext.do(cmd.Context(), apiCommandDeps{httpClient: http.DefaultClient}, apiRequest{
				Method: http.MethodPost,
				Path:   "projects/" + urlPathEscape(projectID) + "/skills/import",
				Body:   body,
			})
			if err != nil {
				return err
			}
			return writeAPIOutput(cmd.OutOrStdout(), response.Body, output)
		},
	}

	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLIFlagNormalization(command.Flags())
	bindAPICommandFlags(command.Flags(), &apiOptions)
	bindAPIOutputFlags(command.Flags(), &output)
	command.Flags().StringVar(&nameOverride, "name", "", "Override the skill name. Must match SKILL.md frontmatter.")
	command.Flags().BoolVar(&enableNow, "enable", false, "Enable the imported skill immediately.")
	command.Flags().BoolVar(&disableNow, "disable", false, "Import the skill in a disabled state.")
	return markCLICommandAPICoverageSpec(command, spec)
}

func buildSkillImportPayload(dir string, nameOverride string, enabled *bool) (skillImportPayload, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return skillImportPayload{}, fmt.Errorf("resolve skill import dir: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return skillImportPayload{}, fmt.Errorf("stat skill import dir: %w", err)
	}
	if !info.IsDir() {
		return skillImportPayload{}, fmt.Errorf("skill import path must be a directory")
	}

	files := make([]skillImportPayloadFile, 0, 8)
	if err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("skill import does not support symlinks: %s", current)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("skill import only supports regular files: %s", current)
		}

		relative, err := filepath.Rel(root, current)
		if err != nil {
			return fmt.Errorf("resolve skill file path %s: %w", current, err)
		}
		content, err := fs.ReadFile(os.DirFS(root), filepath.ToSlash(relative))
		if err != nil {
			return fmt.Errorf("read skill file %s: %w", current, err)
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat skill file %s: %w", current, err)
		}
		files = append(files, skillImportPayloadFile{
			Path:          filepath.ToSlash(relative),
			ContentBase64: base64.StdEncoding.EncodeToString(content),
			IsExecutable:  entryInfo.Mode()&0o111 != 0,
		})
		return nil
	}); err != nil {
		return skillImportPayload{}, err
	}
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})
	if len(files) == 0 {
		return skillImportPayload{}, fmt.Errorf("skill import directory must contain at least one file")
	}

	name := strings.TrimSpace(nameOverride)
	if name == "" {
		name = filepath.Base(root)
	}
	return skillImportPayload{
		Name:      name,
		IsEnabled: enabled,
		Files:     files,
	}, nil
}

func enableFlagValue(enableNow bool, disableNow bool) *bool {
	switch {
	case enableNow:
		value := true
		return &value
	case disableNow:
		value := false
		return &value
	default:
		return nil
	}
}
