package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BetterAndBetterII/openase/internal/httpapi"
	"github.com/spf13/cobra"
)

func newOpenAPICommand() *cobra.Command {
	var output string

	command := &cobra.Command{
		Use:   "openapi",
		Short: "Generate OpenAPI contract artifacts.",
	}

	command.AddCommand(&cobra.Command{
		Use:   "generate",
		Short: "Generate the OpenAPI contract JSON.",
		RunE: func(_ *cobra.Command, _ []string) error {
			payload, err := httpapi.BuildOpenAPIJSON()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
				return fmt.Errorf("create openapi output directory: %w", err)
			}
			if err := os.WriteFile(output, payload, 0o600); err != nil {
				return fmt.Errorf("write openapi contract: %w", err)
			}
			return nil
		},
	})

	command.PersistentFlags().StringVar(&output, "output", "api/openapi.json", "Path to write the generated OpenAPI JSON.")

	return command
}
