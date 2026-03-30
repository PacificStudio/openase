package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BetterAndBetterII/openase/internal/httpapi"
	"github.com/spf13/cobra"
)

func newOpenAPICommand() *cobra.Command {
	var openAPIOutput string
	var cliContractOutput string

	command := &cobra.Command{
		Use:   "openapi",
		Short: "Generate OpenAPI contract artifacts.",
	}

	generateCommand := &cobra.Command{
		Use:   "generate",
		Short: "Generate the OpenAPI contract JSON.",
		RunE: func(_ *cobra.Command, _ []string) error {
			payload, err := httpapi.BuildOpenAPIJSON()
			if err != nil {
				return err
			}
			return writeOpenAPIArtifact(openAPIOutput, payload, "openapi contract")
		},
	}
	generateCommand.Flags().StringVar(&openAPIOutput, "output", "api/openapi.json", "Path to write the generated OpenAPI JSON.")
	command.AddCommand(generateCommand)

	cliContractCommand := &cobra.Command{
		Use:   "cli-contract",
		Short: "Generate the CLI/OpenAPI contract snapshot.",
		RunE: func(_ *cobra.Command, _ []string) error {
			snapshot, err := commandContractSnapshot()
			if err != nil {
				return err
			}
			payload, err := json.MarshalIndent(snapshot, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal cli contract snapshot: %w", err)
			}
			payload = append(payload, '\n')
			return writeOpenAPIArtifact(cliContractOutput, payload, "cli contract snapshot")
		},
	}
	cliContractCommand.Flags().StringVar(&cliContractOutput, "output", "internal/cli/testdata/openapi_cli_contract.json", "Path to write the generated CLI contract snapshot.")
	command.AddCommand(cliContractCommand)

	return command
}

func writeOpenAPIArtifact(output string, payload []byte, label string) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o750); err != nil {
		return fmt.Errorf("create %s output directory: %w", label, err)
	}
	if err := os.WriteFile(output, payload, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", label, err)
	}
	return nil
}
