package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

const (
	atlasBinary   = ".tools/atlas"
	defaultName   = "bootstrap_schema"
	devDBPort     = 55432
	devDBUser     = "postgres"
	devDBPassword = "postgres"
	devDBName     = "atlasdev"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cwd, err := os.Getwd()
	if err != nil {
		fail("get working directory", err)
	}

	mode := "diff"
	name := defaultName
	if len(os.Args) > 1 && os.Args[1] != "" {
		mode = os.Args[1]
	}
	if mode == "diff" && len(os.Args) > 2 && os.Args[2] != "" {
		name = os.Args[2]
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(devDBPort).
			Username(devDBUser).
			Password(devDBPassword).
			Database(devDBName).
			RuntimePath(filepath.Join(cwd, ".tmp", "embedded-postgres-runtime")).
			BinariesPath(filepath.Join(cwd, ".tmp", "embedded-postgres-binaries")).
			DataPath(filepath.Join(cwd, ".tmp", "embedded-postgres-data")),
	)

	if err := pg.Start(); err != nil {
		fail("start embedded postgres", err)
	}
	defer func() {
		if err := pg.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "stop embedded postgres: %v\n", err)
		}
	}()

	devURL := fmt.Sprintf(
		"postgres://%s:%s@127.0.0.1:%d/%s?sslmode=disable&search_path=public",
		devDBUser,
		devDBPassword,
		devDBPort,
		devDBName,
	)

	args := atlasArgs(mode, name, devURL)
	//nolint:gosec // atlasdiff runs a workspace-local helper binary with fixed arguments
	cmd := exec.CommandContext(ctx, filepath.Join(cwd, atlasBinary), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "ATLAS_NO_UPDATE_NOTIFIER=true")

	goBin := filepath.Join(cwd, ".mise-data", "installs", "go", "1.24.13", "bin")
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s:%s", goBin, os.Getenv("PATH")))

	if err := cmd.Run(); err != nil {
		fail("run atlas command", err)
	}
}

func atlasArgs(mode, name, devURL string) []string {
	switch mode {
	case "apply":
		return []string{
			"migrate",
			"apply",
			"--dir", "file://ent/migrate/migrations",
			"--url", devURL,
		}
	case "diff":
		return []string{
			"migrate",
			"diff",
			name,
			"--dir", "file://ent/migrate/migrations",
			"--to", "ent://ent/schema?dialect=postgres",
			"--dev-url", devURL,
		}
	default:
		fail("parse mode", fmt.Errorf("unsupported mode %q", mode))
		return nil
	}
}

func fail(step string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", step, err)
	os.Exit(1)
}
