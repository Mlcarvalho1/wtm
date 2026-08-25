package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/watch"
)

// Palette — lifted from the Nocturne design system's tokens
// (_ds/.../styles.css) so the terminal UI reads as the same product as the
// prototype, translated to what a terminal can actually render: solid
// colors and text attributes stand in for the browser version's gradients,
// box-shadows, and opacity-pulse keyframes.
var (
	colorBg     = lipgloss.Color("#161826")
	colorText   = lipgloss.Color("#e9e9ed")
	colorDim    = lipgloss.Color("#75798c") // neutral-600
	colorDimmer = lipgloss.Color("#595d6c") // neutral-700
	colorFaint  = lipgloss.Color("#3f424d") // neutral-800
	colorLine   = lipgloss.Color("#292b31") // neutral-900, hairline borders

	colorAccent      = lipgloss.Color("#9184d9")
	colorAccentLight = lipgloss.Color("#d2cefd") // accent-300
	colorAccentDeep  = lipgloss.Color("#5d5294") // accent-700

	colorAdd = colorAccentLight
	colorDel = colorDim

	// Per-line-kind hunk backgrounds — stand-ins for the design's hunkBg
	// alpha tints (add: accent@8%, del: text@3.5%, hunk: accent@5%,
	// context/gap: transparent), pushed noticeably past those percentages:
	// alpha blending against the design's lighter canvas reads clearly at
	// those low opacities, but the same math against this terminal's much
	// darker solid background collapses into near-invisibility — these are
	// picked to stay visibly distinct from colorBg and from each other
	// instead of reproducing the alpha values literally.
	colorDiffAddBg  = lipgloss.Color("#2c2547")
	colorDiffDelBg  = lipgloss.Color("#332e38")
	colorDiffHunkBg = lipgloss.Color("#362a5c")

	// Row highlight backgrounds: solid stand-ins for the design's
	// alpha-blended `style-hover`/selection tints, since a terminal cell has
	// no alpha channel. Selected stays the brightest (cursor position);
	// hover is a subtler step so the two read as distinct when a hovered
	// row isn't also the selected one.
	colorSelectedBg    = lipgloss.Color("#242038")
	colorHoverBg       = lipgloss.Color("#1c1a2a")
	colorHoverAccentBg = lipgloss.Color("#2a2640")
)

var (
	styleError = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleSubtitle = lipgloss.NewStyle().Foreground(colorDim)
	styleHelp     = lipgloss.NewStyle().Foreground(colorDim)
	styleDimmer   = lipgloss.NewStyle().Foreground(colorDimmer)

	stylePrompt = lipgloss.NewStyle().Foreground(colorAccentLight).Bold(true)

	styleAccent      = lipgloss.NewStyle().Foreground(colorAccent)
	styleAccentLight = lipgloss.NewStyle().Foreground(colorAccentLight)
	styleText        = lipgloss.NewStyle().Foreground(colorText)

	styleKicker = lipgloss.NewStyle().Foreground(colorDim).Bold(true) // section labels: AWAITING YOU, WORKTREE, THIS RUN...

	styleSelectedRow = lipgloss.NewStyle().Background(lipgloss.Color("#242038")).Foreground(colorText)

	styleTabActive   = lipgloss.NewStyle().Bold(true).Foreground(colorAccentLight)
	styleTabInactive = lipgloss.NewStyle().Foreground(colorDim)
	styleTabHover    = lipgloss.NewStyle().Foreground(colorText)

	styleBadge = lipgloss.NewStyle().Foreground(colorAccentLight).Border(lipgloss.NormalBorder(), false, false, false, false)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorLine).
			Padding(0, 1)

	styleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorFaint).
			Background(lipgloss.Color("#232532")).
			Padding(1, 2)

	styleButtonPrimary = lipgloss.NewStyle().Foreground(colorAccentLight).Bold(true)
	styleButtonGhost   = lipgloss.NewStyle().Foreground(colorDim)

	styleInputActive = lipgloss.NewStyle().Foreground(colorText)
)

// stateColor returns the dot/label color for a fleet-status state, mirroring
// the design's STATES table (running=accent, needs=text+blink, idle=dim,
// stopped=faint outline).
func stateColor(s watch.State) lipgloss.Color {
	switch s {
	case watch.StateRunning:
		return colorAccent
	case watch.StateNeedsInput:
		return colorText
	case watch.StateIdle:
		return colorDimmer
	default:
		return colorFaint
	}
}

// stateLabel is the lowercase state word used throughout the sidebar/session
// tab, matching the design's copy ("needs input" rather than watch.State's
// title-cased "Needs Input" used by the headless `wtm ls` output).
func stateLabel(s watch.State) string {
	switch s {
	case watch.StateRunning:
		return "running"
	case watch.StateNeedsInput:
		return "needs input"
	case watch.StateIdle:
		return "idle"
	default:
		return "stopped"
	}
}
