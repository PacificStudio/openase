package userservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var _ = logging.DeclareComponent("userservice-launchd-support")

// LaunchdSupport describes the current launchd session prerequisites.
type LaunchdSupport struct {
	Domain string
}

// LaunchdServiceReference describes where a launchd service is or should be addressed.
type LaunchdServiceReference struct {
	Domain string
	Target string
	Loaded bool
}

func CheckLaunchdSupport(ctx context.Context, homeDir string, uid int) (LaunchdSupport, error) {
	return checkLaunchdSupportWithRunner(ctx, homeDir, uid, execCommandRunner{}, exec.LookPath)
}

func ResolveLaunchdService(ctx context.Context, uid int, name provider.ServiceName) (LaunchdServiceReference, error) {
	return resolveLaunchdServiceWithRunner(ctx, uid, name, execCommandRunner{})
}

func checkLaunchdSupportWithRunner(
	ctx context.Context,
	homeDir string,
	uid int,
	runner commandRunner,
	lookPath func(string) (string, error),
) (LaunchdSupport, error) {
	if _, err := lookPath("launchctl"); err != nil {
		return LaunchdSupport{}, fmt.Errorf("launchctl is not installed")
	}
	if err := checkLaunchdInstallPrereqs(homeDir); err != nil {
		return LaunchdSupport{}, err
	}

	domain, err := newLaunchdDomainResolver(uid, runner).resolveAvailableDomain(ctx)
	if err != nil {
		return LaunchdSupport{}, err
	}

	return LaunchdSupport{Domain: domain}, nil
}

func resolveLaunchdServiceWithRunner(
	ctx context.Context,
	uid int,
	name provider.ServiceName,
	runner commandRunner,
) (LaunchdServiceReference, error) {
	return newLaunchdDomainResolver(uid, runner).resolveService(ctx, launchdLabel(name))
}

type launchdDomainResolver struct {
	uid    int
	runner commandRunner
}

func newLaunchdDomainResolver(uid int, runner commandRunner) launchdDomainResolver {
	return launchdDomainResolver{
		uid:    uid,
		runner: runner,
	}
}

func (r launchdDomainResolver) resolveService(ctx context.Context, label string) (LaunchdServiceReference, error) {
	for _, domain := range r.candidateDomains() {
		target := launchdTarget(domain, label)
		err := r.runner.Run(ctx, "launchctl", []string{"print", target}, io.Discard, io.Discard)
		switch {
		case err == nil:
			return LaunchdServiceReference{Domain: domain, Target: target, Loaded: true}, nil
		case isLaunchctlProbeMiss(err):
			continue
		case isLaunchctlMissing(err):
			return LaunchdServiceReference{}, fmt.Errorf("launchctl is not installed")
		default:
			return LaunchdServiceReference{}, fmt.Errorf("probe launchd service %s: %w", target, err)
		}
	}

	domain, err := r.resolveAvailableDomain(ctx)
	if err != nil {
		return LaunchdServiceReference{}, err
	}

	return LaunchdServiceReference{
		Domain: domain,
		Target: launchdTarget(domain, label),
		Loaded: false,
	}, nil
}

func (r launchdDomainResolver) resolveAvailableDomain(ctx context.Context) (string, error) {
	domains := r.candidateDomains()
	for _, domain := range domains {
		err := r.runner.Run(ctx, "launchctl", []string{"print", domain}, io.Discard, io.Discard)
		switch {
		case err == nil:
			return domain, nil
		case isLaunchctlProbeMiss(err):
			continue
		case isLaunchctlMissing(err):
			return "", fmt.Errorf("launchctl is not installed")
		default:
			return "", fmt.Errorf("probe launchd domain %s: %w", domain, err)
		}
	}

	return "", fmt.Errorf(
		"launchd is installed, but no usable domain was found for uid %d; tried %s. this login session is not attached to a supported launchd user domain",
		r.uid,
		strings.Join(domains, ", "),
	)
}

func (r launchdDomainResolver) candidateDomains() []string {
	return []string{
		fmt.Sprintf("gui/%d", r.uid),
		fmt.Sprintf("user/%d", r.uid),
	}
}

func checkLaunchdInstallPrereqs(homeDir string) error {
	trimmedHomeDir := strings.TrimSpace(homeDir)
	if trimmedHomeDir == "" {
		return fmt.Errorf("launchd LaunchAgent installation requires a user home directory")
	}
	if !filepath.IsAbs(trimmedHomeDir) {
		return fmt.Errorf("launchd LaunchAgent home directory %q must be absolute", trimmedHomeDir)
	}

	info, err := os.Stat(trimmedHomeDir)
	if err != nil {
		return fmt.Errorf("launchd LaunchAgent installation requires an accessible home directory %q: %w", trimmedHomeDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("launchd LaunchAgent home path %q is not a directory", trimmedHomeDir)
	}

	return nil
}

func isLaunchctlProbeMiss(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func isLaunchctlMissing(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}

func launchdLabel(name provider.ServiceName) string {
	return "com." + name.String()
}

func launchdTarget(domain string, label string) string {
	return domain + "/" + label
}

func launchdPlistPath(homeDir string, name provider.ServiceName) string {
	return filepath.Join(homeDir, "Library", "LaunchAgents", launchdLabel(name)+".plist")
}
