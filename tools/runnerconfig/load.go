package runnerconfig

import (
	"encoding/json"
	"fmt"
)

func Load(cwd string, fs FS) (Overrides, string, bool, error) {
	fs = fs.withDefaults()
	configPath, err := ResolveConfigPath(cwd, fs)
	if err != nil {
		return Overrides{}, "", false, err
	}
	if configPath == "" {
		return Overrides{}, "", false, nil
	}
	content, err := fs.ReadFile(configPath)
	if err != nil {
		return Overrides{}, "", false, fmt.Errorf("read launcher override file: %w", err)
	}
	var overrides Overrides
	if err := json.Unmarshal(content, &overrides); err != nil {
		return Overrides{}, "", false, fmt.Errorf("parse launcher override file: %w", err)
	}
	return overrides, configPath, true, nil
}
