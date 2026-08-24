// Package config loads and saves wtm's small YAML config file:
// the worktree root directory and the set of registered repos.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Repo is a git repository wtm knows how to create worktrees for.
type Repo struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// Config is wtm's on-disk configuration.
type Config struct {
	// WorktreeRoot is the parent directory under which new worktrees are
	// created, as <WorktreeRoot>/<repo-name>/<branch>.
	WorktreeRoot string `yaml:"worktree_root"`
	Repos        []Repo `yaml:"repos"`
}

// Path returns the config file location, honoring $XDG_CONFIG_HOME.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "wtm", "config.yaml"), nil
}

func defaultConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	return Config{WorktreeRoot: filepath.Join(home, "wt")}, nil
}

// Load reads the config file, returning a default config if it doesn't exist yet.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultConfig()
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.WorktreeRoot == "" {
		def, err := defaultConfig()
		if err != nil {
			return Config{}, err
		}
		cfg.WorktreeRoot = def.WorktreeRoot
	}
	return cfg, nil
}

// Save writes the config file, creating its parent directory if needed.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// EnsureRepo registers repo in cfg if not already present (matched by Path),
// returning the possibly-updated config and whether it changed.
func EnsureRepo(cfg Config, repo Repo) (Config, bool) {
	for _, r := range cfg.Repos {
		if r.Path == repo.Path {
			return cfg, false
		}
	}
	cfg.Repos = append(cfg.Repos, repo)
	return cfg, true
}
