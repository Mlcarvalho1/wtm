package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load (default): %v", err)
	}
	if cfg.WorktreeRoot == "" {
		t.Fatalf("expected a default worktree root")
	}

	cfg.Repos = append(cfg.Repos, Repo{Name: "wtm", Path: "/tmp/wtm"})
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Repos) != 1 || loaded.Repos[0].Name != "wtm" {
		t.Fatalf("expected 1 repo named wtm, got %+v", loaded.Repos)
	}
	if loaded.WorktreeRoot != cfg.WorktreeRoot {
		t.Fatalf("expected worktree root %q, got %q", cfg.WorktreeRoot, loaded.WorktreeRoot)
	}
}

func TestEnsureRepo_Dedup(t *testing.T) {
	cfg := Config{}
	cfg, changed := EnsureRepo(cfg, Repo{Name: "a", Path: "/a"})
	if !changed || len(cfg.Repos) != 1 {
		t.Fatalf("expected first EnsureRepo to add repo, got changed=%v repos=%+v", changed, cfg.Repos)
	}

	cfg, changed = EnsureRepo(cfg, Repo{Name: "a", Path: "/a"})
	if changed || len(cfg.Repos) != 1 {
		t.Fatalf("expected duplicate EnsureRepo to be a no-op, got changed=%v repos=%+v", changed, cfg.Repos)
	}
}

func TestPath_RespectsXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := path, filepath.Join(dir, "wtm", "config.yaml"); got != want {
		t.Fatalf("expected path %q, got %q", want, got)
	}
}
