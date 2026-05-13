package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorMajor    = lipgloss.Color("#FF4444")
	colorCritical = lipgloss.Color("#FF00AA")
	colorMinor    = lipgloss.Color("#FFAA00")
	colorPatch    = lipgloss.Color("#44FF88")
	colorBreaking = lipgloss.Color("#FF2222")
	colorHeader  = lipgloss.Color("#5599FF")
	colorMuted   = lipgloss.Color("#888888")
	colorSelected = lipgloss.Color("#334455")
	colorCursor  = lipgloss.Color("#4477AA")
	colorWarning = lipgloss.Color("#FF8800")
	colorGreen   = lipgloss.Color("#44BB66")

	styleCritical = lipgloss.NewStyle().
			Foreground(colorCritical).
			Bold(true)

	styleMajor = lipgloss.NewStyle().
			Foreground(colorMajor).
			Bold(true)

	styleMinor = lipgloss.NewStyle().
			Foreground(colorMinor)

	stylePatch = lipgloss.NewStyle().
			Foreground(colorPatch)

	styleUnknown = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleBreakingBanner = lipgloss.NewStyle().
				Background(colorBreaking).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 1)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorHeader).
			Bold(true)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSelected = lipgloss.NewStyle().
			Background(colorSelected)

	styleCursor = lipgloss.NewStyle().
			Background(colorCursor).
			Foreground(lipgloss.Color("#FFFFFF"))

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#222233")).
			Foreground(lipgloss.Color("#AABBCC")).
			Padding(0, 1)

	styleCheckbox = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	styleHelp = lipgloss.NewStyle().
			Background(lipgloss.Color("#111122")).
			Foreground(lipgloss.Color("#CCDDEE")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorHeader)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorHeader).
			Bold(true).
			Padding(0, 1)

	styleDirTag = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666699")).
			Italic(true)

	styleDepType = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#556677"))

	styleEligibilityBad = lipgloss.NewStyle().
				Foreground(colorWarning)

	styleEligibilityOK = lipgloss.NewStyle().
				Foreground(colorGreen)

	styleScrollIndicator = lipgloss.NewStyle().
				Foreground(colorMuted)

	// Summary line shown below the cursor row in the list view
	styleSummaryLine = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7788AA")).
				Italic(true)

	// Section headers in the detail highlights panel
	styleHighlightHeader = lipgloss.NewStyle().
				Foreground(colorHeader).
				Bold(true)

	styleHighlightBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#334466")).
				Padding(0, 1).
				MarginLeft(1)

	// Red gutter bar painted on every line of a breaking-change section
	styleBreakingMarker = lipgloss.NewStyle().
				Foreground(colorBreaking).
				Bold(true)
)
