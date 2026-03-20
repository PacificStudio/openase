package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/BetterAndBetterII/openase/internal/setup"
	"github.com/spf13/cobra"
)

const (
	defaultSetupHost = "127.0.0.1"
	defaultSetupPort = 19836
)

func newSetupCommand() *cobra.Command {
	var host string
	var port int

	command := &cobra.Command{
		Use:   "setup",
		Short: "Run the first-run setup wizard.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSetupWizard(cmd.Context(), cmd.OutOrStdout(), host, port)
		},
	}

	command.Flags().StringVar(&host, "host", defaultSetupHost, "HTTP listen host.")
	command.Flags().IntVar(&port, "port", defaultSetupPort, "HTTP listen port.")

	return command
}

func runDefaultSetupWizard(ctx context.Context, out io.Writer) error {
	return runSetupWizard(ctx, out, defaultSetupHost, defaultSetupPort)
}

func runSetupWizard(ctx context.Context, out io.Writer, host string, port int) error {
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

	printSetupWizardBanner(out, address)
	_ = openBrowser(address)

	return server.Run(ctx)
}

func printSetupWizardBanner(out io.Writer, address string) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  OpenASE -- 首次启动配置")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  请在浏览器中完成配置: %s\n", address)
	fmt.Fprintln(out, "  浏览器已自动打开。如未打开请手动访问上述地址。")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  等待配置完成...")
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
