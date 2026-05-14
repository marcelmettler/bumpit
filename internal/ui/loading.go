package ui

import (
	"strings"
)

// renderLoadingScreen shows a title, an indeterminate progress bar, and a
// status label. The comet (bright block with fading trail) bounces left-right.
func renderLoadingScreen(m *Model, label string) string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	// 2-char margin on each side.
	trackW := w - 4
	if trackW < 20 {
		trackW = 20
	}

	// Comet shape: ▒▒▓▓████  (tail → head, 8 chars wide)
	const cometLen = 8
	range_ := trackW - cometLen
	if range_ < 1 {
		range_ = 1
	}

	// Move 2 columns per 80 ms frame → one full bounce ≈ 3 s on a wide terminal.
	frame := m.spinnerFrame * 2
	cycle := range_ * 2
	pos := frame % cycle
	if pos > range_ {
		pos = cycle - pos
	}

	bar := make([]rune, trackW)
	for i := range bar {
		bar[i] = '░'
	}
	comet := []rune{'▒', '▒', '▓', '▓', '█', '█', '█', '█'}
	for i, ch := range comet {
		if pos+i < trackW {
			bar[pos+i] = ch
		}
	}

	// Vertical centering: title + blank + bar + blank + label = 5 lines.
	topPad := (h - 5) / 2
	if topPad < 2 {
		topPad = 2
	}

	var sb strings.Builder
	for i := 0; i < topPad; i++ {
		sb.WriteByte('\n')
	}
	sb.WriteString(styleHeader.Render("  chorekit") + "\n\n")
	sb.WriteString(styleMuted.Render("  " + string(bar)) + "\n\n")
	sb.WriteString(styleMuted.Render("  " + label) + "\n")

	return sb.String()
}
