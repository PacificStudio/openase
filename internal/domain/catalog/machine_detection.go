package catalog

import "fmt"

func MachineDetectionMessage(
	detectedOS MachineDetectedOS,
	detectedArch MachineDetectedArch,
	status MachineDetectionStatus,
) string {
	osLabel := machineDetectedOSLabel(detectedOS)
	archLabel := machineDetectedArchLabel(detectedArch)

	switch status {
	case MachineDetectionStatusOK:
		switch {
		case detectedOS != MachineDetectedOSUnknown && detectedArch != MachineDetectedArchUnknown:
			return fmt.Sprintf("Detected %s on %s.", archLabel, osLabel)
		case detectedOS != MachineDetectedOSUnknown:
			return fmt.Sprintf("Detected %s, but the CPU architecture still needs manual confirmation.", osLabel)
		case detectedArch != MachineDetectedArchUnknown:
			return fmt.Sprintf("Detected %s, but the operating system still needs manual confirmation.", archLabel)
		default:
			return "Machine detection completed, but the operating system and architecture still need manual confirmation."
		}
	case MachineDetectionStatusDegraded:
		switch {
		case detectedOS != MachineDetectedOSUnknown && detectedArch == MachineDetectedArchUnknown:
			return fmt.Sprintf("Detected %s, but the CPU architecture could not be confirmed reliably. Continue saving and verify it manually.", osLabel)
		case detectedOS == MachineDetectedOSUnknown && detectedArch != MachineDetectedArchUnknown:
			return fmt.Sprintf("Detected %s, but the operating system could not be confirmed reliably. Continue saving and verify it manually.", archLabel)
		default:
			return "OpenASE could not reliably confirm the operating system and architecture. Continue saving and verify them manually."
		}
	case MachineDetectionStatusPending:
		return "System detection has not run yet. You can keep configuring the machine and verify the platform after a connection test."
	case MachineDetectionStatusUnknown:
		fallthrough
	default:
		return "Operating system and architecture are still unknown. You can keep configuring the machine and verify the platform manually."
	}
}

func machineDetectedOSLabel(value MachineDetectedOS) string {
	switch value {
	case MachineDetectedOSDarwin:
		return "macOS"
	case MachineDetectedOSLinux:
		return "Linux"
	default:
		return "unknown OS"
	}
}

func machineDetectedArchLabel(value MachineDetectedArch) string {
	switch value {
	case MachineDetectedArchAMD64:
		return "amd64"
	case MachineDetectedArchARM64:
		return "arm64"
	default:
		return "unknown architecture"
	}
}
