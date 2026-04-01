package cli

import (
	"log/slog"
	"os"
	"sort"
)

func logConfigLoadFailure(configFile string, overrides map[string]any, err error) {
	if err == nil {
		return
	}

	overrideKeys := make([]string, 0, len(overrides))
	for key := range overrides {
		overrideKeys = append(overrideKeys, key)
	}
	sort.Strings(overrideKeys)

	slog.New(slog.NewTextHandler(os.Stderr, nil)).Error(
		"openase config load failed",
		"operation", "load_config",
		"config_file", configFile,
		"override_keys", overrideKeys,
		"error", err,
	)
}
