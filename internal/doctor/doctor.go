package doctor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/localdiag"
	// Register the pgx database/sql driver used by doctor connectivity checks.
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
)

type Result struct {
	Name    string
	Status  Status
	Summary string
	Detail  string
	Fix     string
}

type Report struct {
	Results []Result
}

type Options struct {
	ConfigFile   string
	RepoRoot     string
	HomeDir      string
	LookPath     func(string) (string, error)
	RunCommand   func(context.Context, string, ...string) (string, error)
	PingDatabase func(context.Context, string) error
}

type loadedConfig struct {
	config config.Config
	result Result
	ok     bool
}

func Diagnose(ctx context.Context, opts Options) Report {
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	runCommand := opts.RunCommand
	if runCommand == nil {
		runCommand = runExecCommand
	}

	pingDatabase := opts.PingDatabase
	if pingDatabase == nil {
		pingDatabase = pingPostgres
	}

	homeDir, homeErr := resolveHomeDir(opts.HomeDir)
	repoRoot, _ := resolveRepoRoot(opts.RepoRoot)

	results := make([]Result, 0, 8)
	cfg := diagnoseConfig(opts.ConfigFile, repoRoot, homeDir)
	results = append(results, cfg.result)
	results = append(results, diagnoseCommands(ctx, lookPath, runCommand)...)
	results = append(results, diagnosePostgres(ctx, cfg, pingDatabase))

	layoutResult := diagnoseOpenASELayout(homeDir, homeErr)
	results = append(results, layoutResult)

	return Report{Results: results}
}

func (r Report) WarningCount() int {
	count := 0
	for _, result := range r.Results {
		if result.Status == StatusWarning {
			count++
		}
	}
	return count
}

func (r Report) ErrorCount() int {
	count := 0
	for _, result := range r.Results {
		if result.Status == StatusError {
			count++
		}
	}
	return count
}

func (r Report) HasErrors() bool {
	return r.ErrorCount() > 0
}

func (r Report) Render() string {
	var builder strings.Builder
	builder.WriteString("🔍 OpenASE 环境诊断\n\n")

	for _, result := range r.Results {
		builder.WriteString("  ")
		builder.WriteString(statusIcon(result.Status))
		builder.WriteString(" ")
		builder.WriteString(result.Name)
		if result.Summary != "" {
			builder.WriteString(" ")
			builder.WriteString(result.Summary)
		}
		builder.WriteString("\n")

		if result.Detail != "" {
			for _, line := range strings.Split(result.Detail, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				builder.WriteString("     ")
				builder.WriteString(line)
				builder.WriteString("\n")
			}
		}
		if result.Fix != "" {
			builder.WriteString("     → 修复: ")
			builder.WriteString(result.Fix)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n")
	_, _ = fmt.Fprintf(&builder, "总结: %d 个警告，%d 个错误\n", r.WarningCount(), r.ErrorCount())
	return builder.String()
}

func diagnoseConfig(configFile string, repoRoot string, homeDir string) loadedConfig {
	path, err := resolveConfigPath(configFile, repoRoot, homeDir)
	if err != nil {
		return loadedConfig{
			result: Result{
				Name:    "配置",
				Status:  StatusError,
				Summary: "无法解析配置文件",
				Detail:  err.Error(),
			},
		}
	}

	loadOptions := config.LoadOptions{}
	if path != "" {
		loadOptions.ConfigFile = path
	}

	cfg, err := config.Load(loadOptions)
	if err != nil {
		return loadedConfig{
			result: Result{
				Name:    "配置",
				Status:  StatusError,
				Summary: "配置加载失败",
				Detail:  err.Error(),
			},
		}
	}

	detail := fmt.Sprintf("server.mode=%s, event.driver=%s", cfg.Server.Mode, cfg.Event.Driver)
	if path == "" {
		return loadedConfig{
			config: cfg,
			ok:     true,
			result: Result{
				Name:    "配置",
				Status:  StatusOK,
				Summary: "使用默认值和环境变量",
				Detail:  detail,
			},
		}
	}

	return loadedConfig{
		config: cfg,
		ok:     true,
		result: Result{
			Name:    "配置",
			Status:  StatusOK,
			Summary: fmt.Sprintf("已加载 %s", path),
			Detail:  detail,
		},
	}
}

func diagnoseCommands(
	ctx context.Context,
	lookPath func(string) (string, error),
	runCommand func(context.Context, string, ...string) (string, error),
) []Result {
	reports := localdiag.Inspect(ctx, localdiag.SetupCommandSpecs(), localdiag.Options{
		LookPath:   lookPath,
		RunCommand: runCommand,
	})

	results := make([]Result, 0, len(reports))
	for _, report := range reports {
		switch report.Status {
		case localdiag.StatusReady:
			results = append(results, Result{
				Name:    report.Name,
				Status:  StatusOK,
				Summary: report.Version,
				Detail:  report.Path,
			})
		case localdiag.StatusVersionError:
			results = append(results, Result{
				Name:    report.Name,
				Status:  StatusWarning,
				Summary: "已安装，但版本探测失败",
				Detail:  fmt.Sprintf("%s: %s", report.Path, report.Error),
			})
		default:
			results = append(results, Result{
				Name:    report.Name,
				Status:  StatusWarning,
				Summary: "未安装（可选）",
			})
		}
	}

	return results
}

func diagnosePostgres(ctx context.Context, cfg loadedConfig, pingDatabase func(context.Context, string) error) Result {
	if !cfg.ok {
		return Result{
			Name:    "PostgreSQL",
			Status:  StatusWarning,
			Summary: "因配置加载失败而跳过",
		}
	}

	dsn := strings.TrimSpace(cfg.config.Database.DSN)
	if dsn == "" {
		return Result{
			Name:    "PostgreSQL",
			Status:  StatusWarning,
			Summary: "未配置 database.dsn",
		}
	}

	location := summarizeDSN(dsn)
	if err := pingDatabase(ctx, dsn); err != nil {
		return Result{
			Name:    "PostgreSQL",
			Status:  StatusError,
			Summary: fmt.Sprintf("连接失败 (%s)", location),
			Detail:  err.Error(),
		}
	}

	return Result{
		Name:    "PostgreSQL",
		Status:  StatusOK,
		Summary: fmt.Sprintf("已连接 (%s)", location),
	}
}

func diagnoseOpenASELayout(homeDir string, homeErr error) Result {
	if homeErr != nil {
		return Result{
			Name:    "~/.openase",
			Status:  StatusError,
			Summary: "无法解析用户目录",
			Detail:  homeErr.Error(),
		}
	}

	baseDir := filepath.Join(homeDir, ".openase")
	missing := make([]string, 0, 3)
	details := make([]string, 0, 3)

	if info, err := os.Stat(baseDir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, "~/.openase")
		} else {
			return Result{
				Name:    "~/.openase",
				Status:  StatusError,
				Summary: "无法检查目录",
				Detail:  err.Error(),
			}
		}
	} else if !info.IsDir() {
		return Result{
			Name:    "~/.openase",
			Status:  StatusError,
			Summary: "路径存在但不是目录",
			Detail:  baseDir,
		}
	}

	envPath := filepath.Join(baseDir, ".env")
	if info, err := os.Stat(envPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, "~/.openase/.env")
		} else {
			return Result{
				Name:    "~/.openase",
				Status:  StatusError,
				Summary: "无法检查 .env 文件",
				Detail:  err.Error(),
			}
		}
	} else {
		mode := info.Mode().Perm()
		if mode != 0o600 {
			details = append(details, fmt.Sprintf(".env 权限是 %04o，期望 0600", mode))
		}
	}

	logsPath := filepath.Join(baseDir, "logs")
	if info, err := os.Stat(logsPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, "~/.openase/logs")
		} else {
			return Result{
				Name:    "~/.openase",
				Status:  StatusError,
				Summary: "无法检查日志目录",
				Detail:  err.Error(),
			}
		}
	} else if !info.IsDir() {
		details = append(details, "~/.openase/logs 不是目录")
	}

	workspacesPath := filepath.Join(baseDir, "workspaces")
	if info, err := os.Stat(workspacesPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, "~/.openase/workspaces")
		} else {
			return Result{
				Name:    "~/.openase",
				Status:  StatusError,
				Summary: "无法检查工作区目录",
				Detail:  err.Error(),
			}
		}
	} else if !info.IsDir() {
		details = append(details, "~/.openase/workspaces 不是目录")
	}

	if len(missing) == 0 && len(details) == 0 {
		return Result{
			Name:    "~/.openase",
			Status:  StatusOK,
			Summary: "目录布局完整",
			Detail:  baseDir,
		}
	}

	fixes := make([]string, 0, 2)
	if len(missing) > 0 {
		fixes = append(fixes, "mkdir -p ~/.openase/logs ~/.openase/workspaces && touch ~/.openase/.env")
	}
	if len(details) > 0 {
		fixes = append(fixes, "chmod 600 ~/.openase/.env")
	}

	detailLines := make([]string, 0, len(missing)+len(details))
	if len(missing) > 0 {
		detailLines = append(detailLines, "缺失: "+strings.Join(missing, ", "))
	}
	detailLines = append(detailLines, details...)

	return Result{
		Name:    "~/.openase",
		Status:  StatusWarning,
		Summary: "目录布局不完整",
		Detail:  strings.Join(detailLines, "\n"),
		Fix:     strings.Join(fixes, " && "),
	}
}

func resolveHomeDir(homeDir string) (string, error) {
	if strings.TrimSpace(homeDir) != "" {
		return homeDir, nil
	}
	return os.UserHomeDir()
}

func resolveRepoRoot(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) != "" {
		return repoRoot, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current working directory: %w", err)
	}

	current := cwd
	for {
		if _, statErr := os.Stat(filepath.Join(current, ".git")); statErr == nil {
			return current, nil
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return "", fmt.Errorf("inspect repository root: %w", statErr)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return cwd, fmt.Errorf("could not find git repository root from %s", cwd)
		}
		current = parent
	}
}

func resolveConfigPath(explicit string, repoRoot string, homeDir string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		absolutePath, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve config path %q: %w", explicit, err)
		}
		info, err := os.Stat(absolutePath)
		if err != nil {
			return "", fmt.Errorf("stat config path %q: %w", absolutePath, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("config path %q must be a file", absolutePath)
		}
		return absolutePath, nil
	}

	candidates := configCandidates(repoRoot, homeDir)
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("stat config candidate %q: %w", candidate, err)
		}
		if info.IsDir() {
			continue
		}
		return candidate, nil
	}

	return "", nil
}

func configCandidates(repoRoot string, homeDir string) []string {
	candidates := make([]string, 0, 8)
	for _, extension := range []string{"yaml", "yml", "json", "toml"} {
		if repoRoot != "" {
			candidates = append(candidates, filepath.Join(repoRoot, "config."+extension))
		}
	}
	for _, extension := range []string{"yaml", "yml", "json", "toml"} {
		if homeDir != "" {
			candidates = append(candidates, filepath.Join(homeDir, ".openase", "config."+extension))
		}
	}
	return candidates
}

func summarizeDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "无法解析 DSN"
	}

	host := parsed.Host
	database := strings.TrimPrefix(parsed.Path, "/")
	if host == "" && database == "" {
		return "DSN 已配置"
	}
	if database == "" {
		return host
	}
	return host + "/" + database
}

func statusIcon(status Status) string {
	switch status {
	case StatusOK:
		return "✅"
	case StatusWarning:
		return "⚠️"
	default:
		return "❌"
	}
}

func runExecCommand(ctx context.Context, name string, args ...string) (string, error) {
	//nolint:gosec // doctor intentionally executes resolved local diagnostics commands
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func pingPostgres(ctx context.Context, dsn string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres connection: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}
