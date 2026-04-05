package containerharness

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const (
	containerSuiteEnv        = "OPENASE_RUN_REMOTE_RUNTIME_CONTAINER_TESTS"
	composeFileEnv           = "OPENASE_TEST_REMOTE_RUNTIME_COMPOSE_FILE"
	artifactDirEnv           = "OPENASE_REMOTE_RUNTIME_ARTIFACT_DIR"
	openaseBinaryEnv         = "OPENASE_TEST_OPENASE_BINARY"
	defaultArtifactDirectory = ".artifacts/remote-runtime-container"
)

type Options struct {
	ComposeFile string
	ArtifactDir string
	ProjectName string
	Env         map[string]string
}

type Project struct {
	dockerPath   string
	composeFile  string
	artifactDir  string
	projectName  string
	repoRoot     string
	baseEnv      map[string]string
	startedNames []string
}

func RequireContainerSuite(t testing.TB) {
	t.Helper()

	if runtime.GOOS != "linux" {
		t.Skip("remote runtime container suite currently requires linux")
	}
	if strings.TrimSpace(os.Getenv(containerSuiteEnv)) != "1" {
		t.Skipf("set %s=1 to run remote runtime container tests", containerSuiteEnv)
	}
}

func RepoRoot(t testing.TB) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve container harness helper path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func DefaultComposeFile(t testing.TB) string {
	t.Helper()

	if value := strings.TrimSpace(os.Getenv(composeFileEnv)); value != "" {
		return filepath.Clean(value)
	}
	return filepath.Join(RepoRoot(t), "scripts", "ci", "remote_runtime_container.compose.yml")
}

func ArtifactRoot(t testing.TB) string {
	t.Helper()

	if value := strings.TrimSpace(os.Getenv(artifactDirEnv)); value != "" {
		return filepath.Clean(value)
	}
	return filepath.Join(RepoRoot(t), defaultArtifactDirectory)
}

func BuiltOpenASEBinary(t testing.TB) string {
	t.Helper()

	if value := strings.TrimSpace(os.Getenv(openaseBinaryEnv)); value != "" {
		// #nosec G703 -- test-only helper accepts an explicit local binary path from the harness env.
		info, err := os.Stat(value)
		if err != nil {
			t.Fatalf("stat %s: %v", value, err)
		}
		if info.IsDir() {
			t.Fatalf("%s is a directory, want executable binary", value)
		}
		return filepath.Clean(value)
	}

	candidate := filepath.Join(RepoRoot(t), "bin", "openase")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		t.Skipf("OpenASE binary is unavailable; build %s or set %s", candidate, openaseBinaryEnv)
	}
	return candidate
}

func FreeTCPPort(t testing.TB) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free tcp port: %v", err)
	}
	defer func() { _ = listener.Close() }()

	return listener.Addr().(*net.TCPAddr).Port
}

func WaitForTCPPort(t testing.TB, address string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("tcp address %s did not become reachable: %v", address, lastErr)
}

func NewProject(t testing.TB, options Options) *Project {
	t.Helper()

	dockerPath := requireDockerCompose(t)
	composeFile := strings.TrimSpace(options.ComposeFile)
	if composeFile == "" {
		composeFile = DefaultComposeFile(t)
	}
	if _, err := os.Stat(composeFile); err != nil {
		t.Fatalf("stat compose file %s: %v", composeFile, err)
	}

	artifactDir := strings.TrimSpace(options.ArtifactDir)
	if artifactDir == "" {
		artifactDir = ArtifactRoot(t)
	}
	if err := os.MkdirAll(artifactDir, 0o750); err != nil {
		t.Fatalf("create artifact dir %s: %v", artifactDir, err)
	}

	projectName := strings.TrimSpace(options.ProjectName)
	if projectName == "" {
		projectName = "openase-" + strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", ""))
	}

	project := &Project{
		dockerPath:  dockerPath,
		composeFile: filepath.Clean(composeFile),
		artifactDir: filepath.Clean(artifactDir),
		projectName: projectName,
		repoRoot:    RepoRoot(t),
		baseEnv:     cloneEnv(options.Env),
	}
	t.Cleanup(func() {
		project.dumpAllLogs(t)
		project.Down(t)
	})
	return project
}

func (p *Project) Up(t testing.TB, extraEnv map[string]string, services ...string) {
	t.Helper()

	if len(services) == 0 {
		t.Fatal("compose up requires at least one service")
	}
	if _, err := p.runCommand(context.Background(), extraEnv, append([]string{"up", "-d"}, services...)...); err != nil {
		t.Fatalf("compose up %v: %v", services, err)
	}
	p.startedNames = appendUnique(p.startedNames, services...)
}

func (p *Project) Down(t testing.TB) {
	t.Helper()

	if p == nil {
		return
	}
	_, _ = p.runCommand(context.Background(), nil, "down", "--remove-orphans", "-v")
}

func (p *Project) Logs(t testing.TB, extraEnv map[string]string, services ...string) string {
	t.Helper()

	args := make([]string, 0, 3+len(services))
	args = append(args, "logs", "--no-color", "--timestamps")
	args = append(args, services...)
	output, err := p.runCommand(context.Background(), extraEnv, args...)
	if err != nil {
		return strings.TrimSpace(string(output) + "\n" + err.Error())
	}
	return strings.TrimSpace(string(output))
}

func (p *Project) WriteLogs(t testing.TB, name string, extraEnv map[string]string, services ...string) string {
	t.Helper()

	if p == nil {
		return ""
	}
	logs := p.Logs(t, extraEnv, services...)
	target := filepath.Join(p.artifactDir, sanitizeArtifactName(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("create artifact parent for %s: %v", target, err)
	}
	if err := os.WriteFile(target, []byte(logs+"\n"), 0o600); err != nil {
		t.Fatalf("write compose logs %s: %v", target, err)
	}
	return target
}

func (p *Project) dumpAllLogs(t testing.TB) {
	t.Helper()

	for _, service := range p.startedNames {
		p.WriteLogs(t, service+".log", nil, service)
	}
}

func (p *Project) runCommand(ctx context.Context, extraEnv map[string]string, args ...string) ([]byte, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	commandArgs := make([]string, 0, 5+len(args))
	commandArgs = append(commandArgs, "compose", "-f", p.composeFile, "-p", p.projectName)
	commandArgs = append(commandArgs, args...)

	// #nosec G204 -- container test helper invokes a validated local docker CLI with controlled compose arguments.
	command := exec.CommandContext(ctxWithTimeout, p.dockerPath, commandArgs...)
	command.Env = mergedEnv(os.Environ(), p.baseEnv, extraEnv)
	command.Dir = p.repoRoot
	output, err := command.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func requireDockerCompose(t testing.TB) string {
	t.Helper()

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker is not available on PATH")
	}

	checkCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// #nosec G204 -- helper probes the local docker compose plugin through a validated docker CLI path.
	output, err := exec.CommandContext(checkCtx, dockerPath, "compose", "version").CombinedOutput()
	if err != nil {
		t.Skipf("docker compose is unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	return dockerPath
}

func cloneEnv(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func mergedEnv(base []string, maps ...map[string]string) []string {
	merged := make(map[string]string, len(base))
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		merged[key] = value
	}
	for _, input := range maps {
		for key, value := range input {
			merged[key] = value
		}
	}

	result := make([]string, 0, len(merged))
	for key, value := range merged {
		result = append(result, key+"="+value)
	}
	return result
}

func appendUnique(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}

func sanitizeArtifactName(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return replacer.Replace(strings.TrimSpace(name))
}
