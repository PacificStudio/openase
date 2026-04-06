package machineprobe

import (
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/logging"
)

var _ = logging.DeclareComponent("machine-probe")

// NormalizePlatform maps raw OS/arch values into the domain detection model.
func NormalizePlatform(rawOS string, rawArch string) (domain.MachineDetectedOS, domain.MachineDetectedArch, domain.MachineDetectionStatus) {
	detectedOS := normalizeDetectedOS(rawOS)
	detectedArch := normalizeDetectedArch(rawArch)
	switch {
	case detectedOS != domain.MachineDetectedOSUnknown && detectedArch != domain.MachineDetectedArchUnknown:
		return detectedOS, detectedArch, domain.MachineDetectionStatusOK
	case detectedOS != domain.MachineDetectedOSUnknown || detectedArch != domain.MachineDetectedArchUnknown:
		return detectedOS, detectedArch, domain.MachineDetectionStatusDegraded
	default:
		return detectedOS, detectedArch, domain.MachineDetectionStatusUnknown
	}
}

// DetectPlatformFromProbeOutput parses the shared `whoami && hostname && uname -srm`
// probe output used by SSH and websocket runtime transports.
func DetectPlatformFromProbeOutput(output string) (domain.MachineDetectedOS, domain.MachineDetectedArch, domain.MachineDetectionStatus) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		return domain.MachineDetectedOSUnknown, domain.MachineDetectedArchUnknown, domain.MachineDetectionStatusUnknown
	}
	fields := strings.Fields(strings.TrimSpace(lines[2]))
	if len(fields) == 0 {
		return domain.MachineDetectedOSUnknown, domain.MachineDetectedArchUnknown, domain.MachineDetectionStatusUnknown
	}
	arch := ""
	if len(fields) > 1 {
		arch = fields[len(fields)-1]
	}
	return NormalizePlatform(fields[0], arch)
}

func normalizeDetectedOS(raw string) domain.MachineDetectedOS {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "darwin", "macos", "mac", "macosx":
		return domain.MachineDetectedOSDarwin
	case "linux":
		return domain.MachineDetectedOSLinux
	default:
		return domain.MachineDetectedOSUnknown
	}
}

func normalizeDetectedArch(raw string) domain.MachineDetectedArch {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "x86_64", "amd64":
		return domain.MachineDetectedArchAMD64
	case "aarch64", "arm64", "arm64e":
		return domain.MachineDetectedArchARM64
	default:
		return domain.MachineDetectedArchUnknown
	}
}
