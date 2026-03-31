package cli

import (
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the local OpenASE environment.",
		Long: strings.TrimSpace(`
Diagnose the local OpenASE environment.

This command checks configuration, runtime dependencies, and local setup health
and prints a human-readable report. It exits non-zero when blocking problems
are found.
`),
		Example: "openase doctor\nopenase doctor --config ~/.openase/config.yaml",
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := doctor.Diagnose(cmd.Context(), doctor.Options{
				ConfigFile: options.configFile,
			})
			if _, err := fmt.Fprint(cmd.OutOrStdout(), report.Render()); err != nil {
				return err
			}
			if report.HasErrors() {
				return newExitError(1, "")
			}
			return nil
		},
	}
}
