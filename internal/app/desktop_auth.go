package app

import "strings"

const desktopDisableAuthEnv = "OPENASE_DESKTOP_DISABLE_AUTH"

func desktopHumanAuthDisabled(lookupEnv func(string) string) bool {
	if lookupEnv == nil {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(lookupEnv(desktopDisableAuthEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
