package provider

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func ProbeGeminiCLIRateLimit(
	ctx context.Context,
	processManager AgentCLIProcessManager,
	geminiCommand AgentCLICommand,
	workingDirectory *AbsolutePath,
	environment []string,
	model string,
) (*CLIRateLimit, *time.Time, error) {
	if processManager == nil {
		return nil, nil, fmt.Errorf("gemini rate limit probe process manager must not be nil")
	}
	if geminiCommand == "" {
		return nil, nil, fmt.Errorf("gemini rate limit probe command must not be empty")
	}

	resolvedGeminiCommand, err := exec.LookPath(geminiCommand.String())
	if err != nil {
		return nil, nil, fmt.Errorf("resolve gemini cli command %q: %w", geminiCommand, err)
	}

	nodeCommand, err := ParseAgentCLICommand("node")
	if err != nil {
		return nil, nil, err
	}
	processSpec, err := NewAgentCLIProcessSpec(
		nodeCommand,
		buildGeminiRateLimitProbeArgs(resolvedGeminiCommand, model),
		workingDirectory,
		environment,
	)
	if err != nil {
		return nil, nil, err
	}

	process, err := processManager.Start(ctx, processSpec)
	if err != nil {
		return nil, nil, fmt.Errorf("start gemini rate limit probe: %w", err)
	}
	if stdin := process.Stdin(); stdin != nil {
		if err := stdin.Close(); err != nil {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer stopCancel()
			_ = process.Stop(stopCtx)
			return nil, nil, fmt.Errorf("close gemini rate limit probe stdin: %w", err)
		}
	}

	stdout := process.Stdout()
	stderr := process.Stderr()

	var stdoutBytes []byte
	var stderrBytes []byte
	var stdoutErr error
	var stderrErr error
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		stdoutBytes, stdoutErr = io.ReadAll(stdout)
	}()
	go func() {
		defer readers.Done()
		stderrBytes, stderrErr = io.ReadAll(stderr)
	}()

	waitErr := process.Wait()
	readers.Wait()

	if stdoutErr != nil {
		return nil, nil, fmt.Errorf("read gemini rate limit probe stdout: %w", stdoutErr)
	}
	if stderrErr != nil {
		return nil, nil, fmt.Errorf("read gemini rate limit probe stderr: %w", stderrErr)
	}
	if waitErr != nil {
		return nil, nil, fmt.Errorf(
			"gemini rate limit probe failed: %w%s",
			waitErr,
			formatGeminiProbeStderr(stderrBytes),
		)
	}

	output := strings.TrimSpace(string(stdoutBytes))
	if output == "" {
		return nil, nil, nil
	}

	rateLimit, err := ParseGeminiCLIRateLimit(stdoutBytes)
	if err != nil {
		return nil, nil, err
	}
	if rateLimit == nil {
		return nil, nil, nil
	}

	observedAt := time.Now().UTC()
	return rateLimit, &observedAt, nil
}

func formatGeminiProbeStderr(stderr []byte) string {
	trimmed := strings.TrimSpace(string(stderr))
	if trimmed == "" {
		return ""
	}

	return ": " + trimmed
}

func buildGeminiRateLimitProbeArgs(geminiCommand string, model string) []string {
	return []string{
		"--input-type=module",
		"-e",
		geminiRateLimitProbeScript,
		geminiCommand,
		strings.TrimSpace(model),
	}
}

const geminiRateLimitProbeScript = `import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { pathToFileURL } from 'node:url';

function resolveGeminiDistRoot(commandPath) {
	let resolved = commandPath;
	try {
		resolved = fs.realpathSync(commandPath);
	} catch {}
	let dir = fs.statSync(resolved).isDirectory() ? resolved : path.dirname(resolved);
	for (let depth = 0; depth < 10; depth += 1) {
		const packagePath = path.join(dir, 'package.json');
		if (fs.existsSync(packagePath)) {
			const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
			if (pkg.name === '@google/gemini-cli') {
				return path.basename(dir) === 'dist' ? dir : path.join(dir, 'dist');
			}
		}
		const parent = path.dirname(dir);
		if (parent === dir) {
			break;
		}
		dir = parent;
	}
	throw new Error('gemini package root not found');
}

const geminiCommand = process.argv[1];
const model = process.argv[2] || '';
const distRoot = resolveGeminiDistRoot(geminiCommand);
const settingsModule = await import(pathToFileURL(path.join(distRoot, 'src/config/settings.js')).href);
const configModule = await import(pathToFileURL(path.join(distRoot, 'src/config/config.js')).href);
const authModule = await import(pathToFileURL(path.join(distRoot, 'src/validateNonInterActiveAuth.js')).href);

const settings = settingsModule.loadSettings();
const argv = {
	prompt: 'openase quota probe',
	promptInteractive: undefined,
	acp: false,
	experimentalAcp: false,
	yolo: false,
	approvalMode: 'default',
	allowedTools: undefined,
	allowedMcpServerNames: undefined,
	listExtensions: false,
	extensions: undefined,
	model: model || undefined,
	query: undefined,
	isCommand: false,
	sandbox: undefined,
	debug: false,
	policy: undefined,
	adminPolicy: undefined,
	screenReader: undefined,
	useWriteTodos: undefined,
};

const config = await configModule.loadCliConfig(settings.merged, 'openase-rate-limit-probe', argv, { cwd: process.cwd() });
await config.initialize();
const authType = await authModule.validateNonInteractiveAuth(
	settings.merged.security.auth.selectedType,
	settings.merged.security.auth.useExternal,
	config,
	settings,
);
await config.refreshAuth(authType);
const quota = await config.refreshUserQuota();
process.stdout.write(
	JSON.stringify({
		authType,
		remaining: config.getQuotaRemaining() ?? null,
		limit: config.getQuotaLimit() ?? null,
		resetTime: config.getQuotaResetTime() ?? null,
		buckets: quota?.buckets ?? [],
	}),
);`
