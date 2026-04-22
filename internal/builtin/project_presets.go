package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	presetdomain "github.com/BetterAndBetterII/openase/internal/domain/projectpreset"
)

//go:embed all:project_presets
var builtinProjectPresetFS embed.FS

func ProjectPresets() []presetdomain.Preset {
	return cloneProjectPresets(builtinProjectPresets)
}

func ProjectPresetByKey(key string) (presetdomain.Preset, bool) {
	trimmed := strings.TrimSpace(key)
	for _, item := range builtinProjectPresets {
		if item.Meta.Key == trimmed {
			return item, true
		}
	}
	return presetdomain.Preset{}, false
}

func cloneProjectPresets(items []presetdomain.Preset) []presetdomain.Preset {
	cloned := make([]presetdomain.Preset, 0, len(items))
	for _, item := range items {
		copied := item
		copied.Statuses = append([]presetdomain.Status(nil), item.Statuses...)
		copied.Workflows = append([]presetdomain.Workflow(nil), item.Workflows...)
		copied.ProjectAI.SkillReferences = append([]presetdomain.SkillReference(nil), item.ProjectAI.SkillReferences...)
		cloned = append(cloned, copied)
	}
	return cloned
}

func mustLoadBuiltinProjectPresets() []presetdomain.Preset {
	items, err := loadBuiltinProjectPresets(builtinProjectPresetFS)
	if err != nil {
		panic(fmt.Sprintf("load builtin project presets: %v", err))
	}
	return items
}

func loadBuiltinProjectPresets(root fs.FS) ([]presetdomain.Preset, error) {
	entries, err := fs.ReadDir(root, "project_presets")
	if err != nil {
		return nil, fmt.Errorf("read builtin project presets directory: %w", err)
	}
	items := make([]presetdomain.Preset, 0, len(entries))
	seenKeys := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := "project_presets/" + name
		content, err := fs.ReadFile(root, path)
		if err != nil {
			return nil, fmt.Errorf("read builtin project preset %s: %w", path, err)
		}
		item, err := presetdomain.ParseYAML(path, content)
		if err != nil {
			return nil, err
		}
		if _, exists := seenKeys[item.Meta.Key]; exists {
			return nil, fmt.Errorf("duplicate builtin project preset key %q", item.Meta.Key)
		}
		seenKeys[item.Meta.Key] = struct{}{}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Meta.Name < items[j].Meta.Name
	})
	return items, nil
}

var builtinProjectPresets = mustLoadBuiltinProjectPresets()
