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
	"github.com/Mlcarvalho1/wtm/internal/gitops"
	"github.com/Mlcarvalho1/wtm/internal/session"
	"github.com/Mlcarvalho1/wtm/internal/termpty"
	"github.com/Mlcarvalho1/wtm/internal/tui"
	"github.com/Mlcarvalho1/wtm/internal/version"
)

func main() {
	root := &cobra.Command{
		Use:     "wtm",
		Short:   "Lightweight worktree/agent manager",
		Version: version.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
	root.SetVersionTemplate("wtm v{{.Version}}\n")
	root.AddCommand(lsCmd())
	root.AddCommand(newCmd())
	root.AddCommand(versionCmd())
	root.AddCommand(updateCmd())

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

func newCmd() *cobra.Command {
	var repoName, baseRef, prompt string
	var launch bool

	cmd := &cobra.Command{
		Use:   "new <branch>",
		Short: "Create a worktree (headless), optionally launching claude in it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]
			if prompt != "" && !launch {
				return fmt.Errorf("--prompt requires --launch")
			}

			cfg, err := loadConfigWithCurrentRepo()
			if err != nil {
				return err
			}

			repo, err := resolveRepo(cfg, repoName)
			if err != nil {
				return err
			}

			if baseRef == "" {
				baseRef, _ = gitops.DefaultBaseRef(repo.Path)
			}

			path := filepath.Join(cfg.WorktreeRoot, repo.Name, branch)
			if err := gitops.AddWorktree(repo.Path, path, branch, baseRef, true); err != nil {
				return err
			}
			fmt.Printf("created worktree %s at %s\n", branch, path)

			if !launch {
				return nil
			}

			name := tui.SessionName(repo.Name, branch)
			command := "claude"
			if prompt != "" {
				command = "claude " + shellQuote(prompt)
			}
			if err := session.New(name, path, command); err != nil {
				return err
			}
			fmt.Printf("launched claude in tmux session %s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoName, "repo", "", "registered repo name (default: repo in the current directory)")
	cmd.Flags().StringVar(&baseRef, "base", "", "base ref to branch from (default: auto-detected, see DefaultBaseRef)")
	cmd.Flags().BoolVar(&launch, "launch", false, "immediately launch claude in a tmux session in the new worktree")
	cmd.Flags().StringVar(&prompt, "prompt", "", "initial prompt to seed the claude session with (requires --launch)")

	return cmd
}

// resolveRepo finds the repo to create a worktree in: by --repo name if
// given, otherwise the repo the current directory belongs to.
func resolveRepo(cfg config.Config, name string) (config.Repo, error) {
	if name != "" {
		for _, r := range cfg.Repos {
			if r.Name == name {
				return r, nil
			}
		}
		return config.Repo{}, fmt.Errorf("no registered repo named %q (registered: %s)", name, repoNames(cfg))
	}
	repo, ok := currentGitRepo()
	if !ok {
		return config.Repo{}, fmt.Errorf("not inside a git repo; pass --repo")
	}
	for _, r := range cfg.Repos {
		if r.Path == repo.Path {
			return r, nil
		}
	}
	return repo, nil
}

func repoNames(cfg config.Config) string {
	names := make([]string, len(cfg.Repos))
	for i, r := range cfg.Repos {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}

// shellQuote wraps s in single quotes for safe embedding in the shell
// command string tmux runs a new session's command through, escaping any
// single quotes already in s.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
