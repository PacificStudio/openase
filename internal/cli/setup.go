package cli

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/BetterAndBetterII/openase/internal/setup"
	"github.com/spf13/cobra"
)

func newSetupCommand() *cobra.Command {
	var host string
	var port int

	command := &cobra.Command{
		Use:   "setup",
		Short: "Run the first-run setup wizard.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := setup.NewService(setup.Options{})
			if err != nil {
				return err
			}

			server := setup.NewServer(setup.ServerOptions{
				Host:    host,
				Port:    port,
				Service: service,
			})
			address := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/setup"

			fmt.Fprintf(cmd.OutOrStdout(), "OpenASE setup wizard listening at %s\n", address)
			_ = openBrowser(address)

			return server.Run(cmd.Context())
		},
	}

	command.Flags().StringVar(&host, "host", "127.0.0.1", "HTTP listen host.")
	command.Flags().IntVar(&port, "port", 19836, "HTTP listen port.")

	return command
}

func openBrowser(url string) error {
	var name string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		name = "open"
		args = []string{url}
	default:
		name = "xdg-open"
		args = []string{url}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return exec.CommandContext(ctx, name, args...).Start()
}
