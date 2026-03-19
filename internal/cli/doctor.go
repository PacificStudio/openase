package cli

import (
	"fmt"

	"github.com/BetterAndBetterII/openase/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the local OpenASE environment.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := doctor.Diagnose(cmd.Context(), doctor.Options{
				ConfigFile: options.configFile,
			})
			fmt.Fprint(cmd.OutOrStdout(), report.Render())
			if report.HasErrors() {
				return newExitError(1, "")
			}
			return nil
		},
	}
}
