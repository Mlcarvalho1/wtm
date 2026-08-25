// Package termpty embeds a live tmux client inside wtm's own TUI: it spawns
// `tmux attach-session` under a PTY wtm owns, feeds its output into a virtual
// terminal emulator, and forwards keystrokes back in. Unlike internal/session,
// this package's whole job is parsing the ANSI stream and holding a terminal
// buffer — that boundary is deliberately crossed here, and nowhere else.
package termpty

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"

	"github.com/Mlcarvalho1/wtm/internal/session"
)

// program is the running Bubble Tea program, recorded by Init so the
// goroutines Attach starts can push messages into it. Bubble Tea v1 takes
// its initial model by value, so a *tea.Program field can't be back-filled
// onto the Model after tea.NewProgram returns — this package-level setter is
// the standard workaround for that self-reference need.
var program *tea.Program

// Init records the running program. Call it once, right after
// tea.NewProgram, before p.Run().
func Init(p *tea.Program) {
	program = p
}

// StartedMsg reports the result of an AttachCmd.
type StartedMsg struct {
	Attachment *Attachment
	Err        error
}

// ExitedMsg reports that an Attachment's tmux client exited, whether from a
// detach (Ctrl-b d, or Attachment.Detach) or the session itself ending.
type ExitedMsg struct{ Err error }

// FrameMsg signals that new PTY output was parsed into the screen buffer.
// It carries no data — its only job is to make Bubble Tea call View() again.
type FrameMsg struct{}

const frameInterval = 33 * time.Millisecond

// Attachment is a live PTY-backed `tmux attach-session` client together with
// the virtual terminal emulator rendering its output.
type Attachment struct {
	session string

	ptmx *os.File
	cmd  *exec.Cmd
	emu  *vt.SafeEmulator

	dirty     int32
	done      chan struct{}
	closeOnce sync.Once
}

// AttachCmd runs Attach inside a tea.Cmd, matching this codebase's existing
// convention (loadWorktreesCmd, pollCmd, ...) of doing blocking IO inside a
// command closure rather than directly in Update.
func AttachCmd(sessionName string, cols, rows int) tea.Cmd {
	return func() tea.Msg {
		a, err := Attach(sessionName, cols, rows)
		return StartedMsg{Attachment: a, Err: err}
	}
}

// Attach spawns `tmux attach-session -t sessionName` under a PTY sized
// cols x rows and starts the goroutines that pump its output into a virtual
// terminal emulator and pump encoded key input back into the PTY.
func Attach(sessionName string, cols, rows int) (*Attachment, error) {
	if program == nil {
		return nil, errors.New("termpty: Init must be called before Attach")
	}

	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	if err != nil {
		return nil, err
	}

	a := &Attachment{
		session: sessionName,
		ptmx:    ptmx,
		cmd:     cmd,
		emu:     vt.NewSafeEmulator(cols, rows),
		done:    make(chan struct{}),
	}

	go a.pumpOutput()
	go a.pumpInput()
	go a.frameLoop()
	go a.waitExit()

	return a, nil
}

// pumpOutput feeds PTY output (tmux -> our emulator) until the PTY errors
// (the child exited or was closed), then stops the frame ticker. The actual
// ExitedMsg is sent by waitExit, which carries the real exit status —
// reading a closed PTY just yields a low-level I/O error, not that status.
func (a *Attachment) pumpOutput() {
	buf := make([]byte, 32*1024)
	for {
		n, err := a.ptmx.Read(buf)
		if n > 0 {
			_, _ = a.emu.Write(buf[:n])
			atomic.StoreInt32(&a.dirty, 1)
		}
		if err != nil {
			a.stop()
			return
		}
	}
}

// pumpInput feeds encoded key input (our emulator -> tmux) until the
// emulator is closed, which unblocks this loop's Read with io.EOF.
func (a *Attachment) pumpInput() {
	buf := make([]byte, 4096)
	for {
		n, err := a.emu.Read(buf)
		if n > 0 {
			if _, werr := a.ptmx.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// frameLoop coalesces bursts of PTY output into a steady redraw rate instead
// of sending Bubble Tea one message per PTY read.
func (a *Attachment) frameLoop() {
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if atomic.SwapInt32(&a.dirty, 0) == 1 {
				program.Send(FrameMsg{})
			}
		case <-a.done:
			return
		}
	}
}

func (a *Attachment) waitExit() {
	err := a.cmd.Wait()
	a.stop()
	program.Send(ExitedMsg{Err: err})
}

func (a *Attachment) stop() {
	a.closeOnce.Do(func() { close(a.done) })
}

// Forward encodes a key event the way the attached terminal's current modes
// (application cursor keys, bracketed paste, ...) require and sends it on.
func (a *Attachment) Forward(msg tea.KeyMsg) {
	if msg.Paste {
		a.emu.Paste(string(msg.Runes))
		return
	}
	a.emu.SendKey(toVTKey(msg))
}

// Render returns the current screen as a plain string, sized to exactly the
// emulator's configured cols x rows.
func (a *Attachment) Render() string {
	return a.emu.Render()
}

// Resize reflows both the emulator's screen and the PTY itself so the child
// (tmux, and whatever it's running) sees the new size.
func (a *Attachment) Resize(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	a.emu.Resize(cols, rows)
	_ = pty.Setsize(a.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

// Detach asks tmux to disconnect this client, leaving the session (and
// whatever's running inside it) alive. The mode transition back to the
// normal list view happens when the resulting ExitedMsg arrives, exactly as
// it would for a manual Ctrl-b d.
func (a *Attachment) Detach() error {
	return session.Detach(a.session)
}

// Close releases the PTY and emulator. Safe to call after an ExitedMsg has
// already been received (the child is gone by then; this just frees
// resources) or, defensively, at any other time.
func (a *Attachment) Close() error {
	a.stop()
	_ = a.emu.Close()
	return a.ptmx.Close()
}
