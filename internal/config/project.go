// Package config handles local project configuration (.flotio.yaml).
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig holds project-level settings stored in .flotio.yaml.
type ProjectConfig struct {
	ProjectID int64  `yaml:"project_id"`
	Name      string `yaml:"name,omitempty"`
}

// FindProject walks up from dir looking for .flotio.yaml.
// Returns the config and its directory, or nil if not found.
func FindProject(dir string) (*ProjectConfig, string) {
	for {
		path := filepath.Join(dir, ".flotio.yaml")
		data, err := os.ReadFile(path)
		if err == nil {
			var pc ProjectConfig
			if yaml.Unmarshal(data, &pc) == nil && pc.ProjectID > 0 {
				return &pc, dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ""
		}
		dir = parent
	}
}

// SaveProject writes .flotio.yaml to dir.
func SaveProject(dir string, pc *ProjectConfig) error {
	data, err := yaml.Marshal(pc)
	if err != nil {
		return fmt.Errorf("marshaling project config: %w", err)
	}
	path := filepath.Join(dir, ".flotio.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// ResolveProjectID returns the project ID to use:
// 1. explicit flag, 2. .flotio.yaml (walk up), 3. error.
func ResolveProjectID(flagID int64) (int64, error) {
	if flagID > 0 {
		return flagID, nil
	}
	cwd, _ := os.Getwd()
	pc, dir := FindProject(cwd)
	if pc != nil {
		return pc.ProjectID, nil
	}
	_ = dir
	return 0, fmt.Errorf("no project ID given and no .flotio.yaml found — run 'flotio init' first")
}
