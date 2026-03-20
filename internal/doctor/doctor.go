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
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	// Register the pgx database/sql driver used by doctor connectivity checks.
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.yaml.in/yaml/v3"
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

type hookScriptCheck struct {
	result Result
}

type loadedConfig struct {
	config config.Config
	result Result
	ok     bool
}

type harnessFrontmatter struct {
	WorkflowHooks map[string][]hookCommand `yaml:"workflow_hooks"`
	TicketHooks   map[string][]hookCommand `yaml:"ticket_hooks"`
	Hooks         map[string][]hookCommand `yaml:"hooks"`
}

type hookCommand struct {
	Cmd string `yaml:"cmd"`
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
	repoRoot, repoErr := resolveRepoRoot(opts.RepoRoot)

	results := make([]Result, 0, 8)
	cfg := diagnoseConfig(opts.ConfigFile, repoRoot, homeDir)
	results = append(results, cfg.result)
	results = append(results, diagnoseGit(ctx, lookPath, runCommand))
	results = append(results, diagnoseAgentCLI(ctx, lookPath, runCommand, "claude", "Claude Code"))
	results = append(results, diagnoseAgentCLI(ctx, lookPath, runCommand, "codex", "Codex"))
	results = append(results, diagnosePostgres(ctx, cfg, pingDatabase))

	layoutResult := diagnoseOpenASELayout(homeDir, homeErr)
	results = append(results, layoutResult)

	harnessResult := diagnoseHarnesses(repoRoot, repoErr)
	results = append(results, harnessResult.result)

	hookResult := diagnoseHookScripts(repoRoot, repoErr)
	results = append(results, hookResult.result)

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

func diagnoseGit(ctx context.Context, lookPath func(string) (string, error), runCommand func(context.Context, string, ...string) (string, error)) Result {
	gitPath, err := lookPath("git")
	if err != nil {
		return Result{
			Name:    "Git",
			Status:  StatusError,
			Summary: "未安装或不在 PATH 中",
			Fix:     "安装 Git，并确保 `git` 可执行文件在 PATH 中",
		}
	}

	output, err := runCommand(ctx, gitPath, "--version")
	if err != nil {
		return Result{
			Name:    "Git",
			Status:  StatusError,
			Summary: "已找到可执行文件，但无法读取版本",
			Detail:  err.Error(),
		}
	}

	return Result{
		Name:    "Git",
		Status:  StatusOK,
		Summary: strings.TrimSpace(output),
	}
}

func diagnoseAgentCLI(
	ctx context.Context,
	lookPath func(string) (string, error),
	runCommand func(context.Context, string, ...string) (string, error),
	commandName string,
	displayName string,
) Result {
	path, err := lookPath(commandName)
	if err != nil {
		return Result{
			Name:    displayName,
			Status:  StatusWarning,
			Summary: "未安装（可选）",
		}
	}

	output, err := runCommand(ctx, path, "--version")
	if err != nil {
		return Result{
			Name:    displayName,
			Status:  StatusWarning,
			Summary: "已安装，但版本探测失败",
			Detail:  fmt.Sprintf("%s: %s", path, err),
		}
	}

	versionLine := firstNonEmptyLine(output)
	if versionLine == "" {
		versionLine = path
	}

	return Result{
		Name:    displayName,
		Status:  StatusOK,
		Summary: versionLine,
		Detail:  path,
	}
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
		fixes = append(fixes, "mkdir -p ~/.openase/logs && touch ~/.openase/.env")
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

func diagnoseHarnesses(repoRoot string, repoErr error) hookScriptCheck {
	if repoErr != nil {
		return hookScriptCheck{
			result: Result{
				Name:    "Harness",
				Status:  StatusWarning,
				Summary: "无法定位仓库根目录",
				Detail:  repoErr.Error(),
			},
		}
	}

	harnessRoot := filepath.Join(repoRoot, ".openase", "harnesses")
	count, err := countFiles(harnessRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return hookScriptCheck{
				result: Result{
					Name:    "Harness",
					Status:  StatusWarning,
					Summary: "未找到 .openase/harnesses 目录",
					Fix:     "在仓库中创建 `.openase/harnesses/` 并提交至少一个 Harness 文件",
				},
			}
		}
		return hookScriptCheck{
			result: Result{
				Name:    "Harness",
				Status:  StatusError,
				Summary: "扫描 Harness 目录失败",
				Detail:  err.Error(),
			},
		}
	}

	if count == 0 {
		return hookScriptCheck{
			result: Result{
				Name:    "Harness",
				Status:  StatusWarning,
				Summary: "未找到 Harness 文件",
				Detail:  harnessRoot,
			},
		}
	}

	return hookScriptCheck{
		result: Result{
			Name:    "Harness",
			Status:  StatusOK,
			Summary: fmt.Sprintf("已加载 %d 个 Harness 文件", count),
			Detail:  harnessRoot,
		},
	}
}

func diagnoseHookScripts(repoRoot string, repoErr error) hookScriptCheck {
	if repoErr != nil {
		return hookScriptCheck{
			result: Result{
				Name:    "Hook 脚本",
				Status:  StatusWarning,
				Summary: "因仓库根目录不可用而跳过",
				Detail:  repoErr.Error(),
			},
		}
	}

	references, parseErrors, err := collectHookScriptReferences(repoRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return hookScriptCheck{
				result: Result{
					Name:    "Hook 脚本",
					Status:  StatusWarning,
					Summary: "未找到 Harness，无法分析 Hook 脚本",
				},
			}
		}
		return hookScriptCheck{
			result: Result{
				Name:    "Hook 脚本",
				Status:  StatusError,
				Summary: "扫描 Hook 配置失败",
				Detail:  err.Error(),
			},
		}
	}

	if len(references) == 0 {
		detail := strings.Join(parseErrors, "\n")
		return hookScriptCheck{
			result: Result{
				Name:    "Hook 脚本",
				Status:  StatusWarning,
				Summary: "未发现被 Harness 引用的脚本",
				Detail:  detail,
			},
		}
	}

	missing := make([]string, 0)
	notExecutable := make([]string, 0)
	healthy := make([]string, 0, len(references))

	for _, relativePath := range references {
		absolutePath := filepath.Join(repoRoot, filepath.FromSlash(relativePath))
		info, statErr := os.Stat(absolutePath)
		if statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) {
				missing = append(missing, relativePath)
				continue
			}
			return hookScriptCheck{
				result: Result{
					Name:    "Hook 脚本",
					Status:  StatusError,
					Summary: "检查脚本文件失败",
					Detail:  statErr.Error(),
				},
			}
		}
		if info.IsDir() {
			missing = append(missing, relativePath)
			continue
		}
		if info.Mode().Perm()&0o111 == 0 {
			notExecutable = append(notExecutable, relativePath)
			continue
		}
		healthy = append(healthy, relativePath)
	}

	detailLines := append([]string{}, parseErrors...)
	if len(missing) > 0 {
		detailLines = append(detailLines, "缺失: "+strings.Join(missing, ", "))
	}
	if len(notExecutable) > 0 {
		detailLines = append(detailLines, "缺少执行权限: "+strings.Join(notExecutable, ", "))
	}

	switch {
	case len(missing) > 0:
		fixes := make([]string, 0, 2)
		fixes = append(fixes, "补齐缺失脚本，或从 Harness 中移除无效引用")
		if len(notExecutable) > 0 {
			fixes = append(fixes, "chmod +x "+strings.Join(notExecutable, " "))
		}
		return hookScriptCheck{
			result: Result{
				Name:    "Hook 脚本",
				Status:  StatusError,
				Summary: fmt.Sprintf("%d 个脚本缺失，%d 个脚本可用", len(missing), len(healthy)),
				Detail:  strings.Join(detailLines, "\n"),
				Fix:     strings.Join(fixes, " && "),
			},
		}
	case len(notExecutable) > 0:
		return hookScriptCheck{
			result: Result{
				Name:    "Hook 脚本",
				Status:  StatusWarning,
				Summary: fmt.Sprintf("已配置 %d 个脚本，其中 %d 个缺少执行权限", len(references), len(notExecutable)),
				Detail:  strings.Join(detailLines, "\n"),
				Fix:     "chmod +x " + strings.Join(notExecutable, " "),
			},
		}
	default:
		return hookScriptCheck{
			result: Result{
				Name:    "Hook 脚本",
				Status:  StatusOK,
				Summary: fmt.Sprintf("已配置 %d 个脚本", len(references)),
				Detail:  strings.Join(healthy, "\n"),
			},
		}
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
			candidates = append(candidates, filepath.Join(repoRoot, "openase."+extension))
		}
	}
	for _, extension := range []string{"yaml", "yml", "json", "toml"} {
		if homeDir != "" {
			candidates = append(candidates, filepath.Join(homeDir, ".openase", "openase."+extension))
		}
	}
	return candidates
}

func countFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func collectHookScriptReferences(repoRoot string) ([]string, []string, error) {
	harnessRoot := filepath.Join(repoRoot, ".openase", "harnesses")
	references := map[string]struct{}{}
	parseErrors := make([]string, 0)

	err := filepath.WalkDir(harnessRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		//nolint:gosec // paths come from walking the already-selected repository root
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read harness file %s: %w", path, readErr)
		}

		commands, commandErr := parseHookCommands(string(content))
		if commandErr != nil {
			relativePath, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				relativePath = path
			}
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %s", filepath.ToSlash(relativePath), commandErr))
			return nil
		}

		for _, command := range commands {
			if reference, ok := parseScriptReference(command); ok {
				references[reference] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	items := make([]string, 0, len(references))
	for item := range references {
		items = append(items, item)
	}
	sort.Strings(items)
	sort.Strings(parseErrors)
	return items, parseErrors, nil
}

func parseHookCommands(content string) ([]string, error) {
	frontmatter, ok := extractFrontmatter(content)
	if !ok {
		return nil, errors.New("缺少 YAML frontmatter")
	}

	var document harnessFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return nil, fmt.Errorf("解析 frontmatter 失败: %w", err)
	}

	commands := make([]string, 0)
	collectHookCommands(document.WorkflowHooks, &commands)
	collectHookCommands(document.TicketHooks, &commands)
	collectHookCommands(document.Hooks, &commands)
	return commands, nil
}

func collectHookCommands(groups map[string][]hookCommand, commands *[]string) {
	for _, items := range groups {
		for _, item := range items {
			command := strings.TrimSpace(item.Cmd)
			if command == "" {
				continue
			}
			*commands = append(*commands, command)
		}
	}
}

func extractFrontmatter(content string) (string, bool) {
	if !strings.HasPrefix(content, "---\n") {
		return "", false
	}

	rest := strings.TrimPrefix(content, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}

	return rest[:end], true
}

func parseScriptReference(command string) (string, bool) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", false
	}

	first := trimShellToken(fields[0])
	if first == "bash" || first == "sh" || first == "zsh" {
		if len(fields) < 2 {
			return "", false
		}
		return normalizeScriptPath(fields[1])
	}

	return normalizeScriptPath(first)
}

func normalizeScriptPath(raw string) (string, bool) {
	token := trimShellToken(raw)
	switch {
	case strings.HasPrefix(token, "./"):
		return strings.TrimPrefix(filepath.ToSlash(token), "./"), true
	case strings.HasPrefix(token, ".openase/"):
		return filepath.ToSlash(token), true
	case strings.HasPrefix(token, "scripts/"):
		return filepath.ToSlash(token), true
	default:
		return "", false
	}
}

func trimShellToken(raw string) string {
	return strings.Trim(raw, "\"'")
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

func firstNonEmptyLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
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
