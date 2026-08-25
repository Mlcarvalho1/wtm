// Package session shells out to tmux for session lifecycle management.
// tmux owns persistence and PTY buffering; this package never parses ANSI
// streams or holds a terminal buffer itself.
package session

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func runTmux(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// Exists reports whether a tmux session with the given name is currently running.
func Exists(name string) bool {
	_, err := runTmux("has-session", "-t", name)
	return err == nil
}

// List returns the names of all currently running tmux sessions.
// It returns an empty slice (not an error) when the tmux server has no sessions.
func List() ([]string, error) {
	out, err := runTmux("list-sessions", "-F", "#{session_name}")
	if err != nil {
		if strings.Contains(err.Error(), "no server running") || strings.Contains(err.Error(), "no current server") {
			return nil, nil
		}
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// New starts a new detached tmux session named `name`, running `command` with
// its working directory set to dir. Fails if a session with that name already exists.
// The status bar is turned off: wtm's own header already shows equivalent
// (and richer) context, so tmux's would just be a redundant line, most
// noticeable when the session is embedded in wtm's own attached pane where
// every row is scarce.
func New(name, dir, command string) error {
	if Exists(name) {
		return fmt.Errorf("session %q already exists", name)
	}
	if _, err := runTmux("new-session", "-d", "-s", name, "-c", dir, command); err != nil {
		return err
	}
	_, err := runTmux("set-option", "-t", name, "status", "off")
	return err
}

// Kill terminates a tmux session. No error if the session doesn't exist.
func Kill(name string) error {
	if !Exists(name) {
		return nil
	}
	_, err := runTmux("kill-session", "-t", name)
	return err
}

// Detach disconnects whatever client(s) are currently attached to a tmux
// session, leaving the session (and the program running inside it) alive.
// A no-op, not an error, if no client is attached.
func Detach(name string) error {
	_, err := runTmux("detach-client", "-s", name)
	return err
}

// CapturePane returns the current visible plain-text contents of a session's pane.
func CapturePane(name string) (string, error) {
	return runTmux("capture-pane", "-p", "-t", name)
}

// AttachArgs returns the argv (excluding the "tmux" binary itself) to attach
// to a session, for use with an exec-and-suspend flow (e.g. Bubble Tea's
// tea.ExecProcess), which needs the full command including the binary name.
func AttachArgs(name string) []string {
	return []string{"tmux", "attach-session", "-t", name}
}
