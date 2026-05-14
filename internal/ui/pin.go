package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcelmettler/chorekit/internal/pkg"
)

func renderPinList(m *Model) string {
	var sb strings.Builder

	count := len(m.pinDeps)
	sb.WriteString(styleTitle.Render("  Unpinned Dependencies") + "\n")
	if count == 0 {
		sb.WriteString(styleEligibilityOK.Render("  All dependencies are pinned to exact versions.") + "\n")
		sb.WriteString(renderPinStatusBar(m))
		return sb.String()
	}
	sb.WriteString(styleMuted.Render(fmt.Sprintf(
		"  %d dependenc%s using ^ or ~ ranges",
		count, pluralIes(count),
	)) + "\n\n")

	sb.WriteString(renderPinColumnHeaders() + "\n")

	start, end := m.pinListWindow(len(m.pinFiltered))
	for i := start; i < end; i++ {
		sb.WriteString(renderPinRow(m.pinFiltered[i], i == m.pinCursor) + "\n")
	}
	if start > 0 {
		sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
	}
	if end < len(m.pinFiltered) {
		sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(m.pinFiltered)-end)) + "\n")
	}

	sb.WriteString("\n")
	if sel := countSelectedPin(m.pinDeps); sel > 0 {
		sb.WriteString(styleCheckbox.Render(fmt.Sprintf("  %d package(s) selected", sel)) + "\n")
	}
	if m.statusMsg != "" {
		sb.WriteString(styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderPinStatusBar(m))
	return sb.String()
}

func renderPinColumnHeaders() string {
	cols := fmt.Sprintf("  %-3s %-34s %-14s %-14s %-8s %s",
		"", "Package", "Current", "Pinned", "Type", "Dir")
	return styleMuted.Render(cols)
}

func renderPinRow(d *pkg.UnpinnedDep, isCursor bool) string {
	check := "[ ]"
	if d.Selected {
		check = styleCheckbox.Render("[x]")
	}
	name := truncate(d.Name, 34)
	current := styleWarning.Render(padRight(d.Version, 14))
	pinned := styleEligibilityOK.Render(padRight(d.Pinned, 14))
	depType := styleDepType.Render(padRight(string(d.DepType), 8))
	dir := styleDirTag.Render("(" + d.DirName + ")")

	row := fmt.Sprintf("  %s %-34s %s %s %s %s", check, name, current, pinned, depType, dir)
	if isCursor {
		return styleCursor.Render(row)
	}
	if d.Selected {
		return styleSelected.Render(row)
	}
	return row
}

func renderPinStatusBar(m *Model) string {
	hints := []string{
		"j/k: navigate",
		"space: select",
		"a: all",
		"p: pin selected",
		"q: quit",
	}
	bar := strings.Join(hints, "  ")
	available := m.width - 2
	if available < 0 {
		available = 80
	}
	if lipgloss.Width(bar) > available {
		bar = "j/k:nav  space:sel  p:pin  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(bar)
}

func renderPinDone(count int, pinned []*pkg.UnpinnedDep) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("\n  Pin Complete") + "\n\n")
	sb.WriteString(styleEligibilityOK.Render(fmt.Sprintf("  Pinned %d dependenc%s:", count, pluralIes(count))) + "\n")

	// Group by file for a cleaner summary.
	byFile := make(map[string][]*pkg.UnpinnedDep)
	var order []string
	for _, d := range pinned {
		if _, seen := byFile[d.File]; !seen {
			order = append(order, d.File)
		}
		byFile[d.File] = append(byFile[d.File], d)
	}
	for _, file := range order {
		sb.WriteString(styleMuted.Render(fmt.Sprintf("\n  %s", file)) + "\n")
		for _, d := range byFile[file] {
			sb.WriteString(styleMuted.Render(fmt.Sprintf(
				"    • %-34s %s → %s",
				d.Name,
				styleWarning.Render(d.Version),
				styleEligibilityOK.Render(d.Pinned),
			)) + "\n")
		}
	}
	sb.WriteString("\n" + styleMuted.Render("  Press any key to quit.") + "\n")
	return sb.String()
}

func countSelectedPin(deps []*pkg.UnpinnedDep) int {
	n := 0
	for _, d := range deps {
		if d.Selected {
			n++
		}
	}
	return n
}

func (m *Model) pinListWindow(total int) (start, end int) {
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.pinScroll
	if m.pinCursor < start {
		start = m.pinCursor
	}
	if m.pinCursor >= start+maxVisible {
		start = m.pinCursor - maxVisible + 1
	}
	m.pinScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}
