package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Mlcarvalho1/wtm/internal/config"
	"github.com/Mlcarvalho1/wtm/internal/termpty"
	"github.com/Mlcarvalho1/wtm/internal/tui"
)

func main() {
	root := &cobra.Command{
		Use:   "wtm",
		Short: "Lightweight worktree/agent manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
	root.AddCommand(lsCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func lsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List worktrees across registered repos (headless)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigWithCurrentRepo()
			if err != nil {
				return err
			}
			rows, errs := tui.LoadWorktrees(cfg)
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, "warning:", e)
			}
			for _, r := range rows {
				fmt.Printf("%s  %-14s %-24s %s\n", r.State.Icon(), r.RepoName, r.Branch, r.Path)
			}
			return nil
		},
	}
}

func runTUI() error {
	cfg, err := loadConfigWithCurrentRepo()
	if err != nil {
		return err
	}
	p := tea.NewProgram(tui.New(cfg), tea.WithMouseAllMotion())
	termpty.Init(p)
	_, err = p.Run()
	return err
}

// loadConfigWithCurrentRepo loads the wtm config and, if the current
// directory is inside a git repo not already registered, adds it.
func loadConfigWithCurrentRepo() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}

	repo, ok := currentGitRepo()
	if !ok {
		return cfg, nil
	}

	updated, changed := config.EnsureRepo(cfg, repo)
	if changed {
		if err := config.Save(updated); err != nil {
			return config.Config{}, err
		}
	}
	return updated, nil
}

func currentGitRepo() (config.Repo, bool) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return config.Repo{}, false
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return config.Repo{}, false
	}
	return config.Repo{Name: filepath.Base(path), Path: path}, true
}
