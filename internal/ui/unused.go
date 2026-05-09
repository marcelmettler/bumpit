package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcelmettler/bumpit/internal/pkg"
)

func renderUnusedList(m *Model) string {
	var sb strings.Builder

	title := styleTitle.Render("  Unused Dependencies")
	count := len(m.unusedPackages)
	var countStr string
	if count == 0 {
		countStr = styleMuted.Render("  No unused direct dependencies found.")
	} else {
		countStr = styleMuted.Render(fmt.Sprintf("  %d unused direct dependenc%s found", count, pluralIes(count)))
	}
	sb.WriteString(title + "\n")
	sb.WriteString(countStr + "\n\n")

	sb.WriteString(renderUnusedColumnHeaders())
	sb.WriteString("\n")

	visible := m.unusedFiltered
	if len(visible) == 0 {
		sb.WriteString(styleMuted.Render("  Nothing to show.") + "\n")
	} else {
		start, end := m.unusedListWindow(len(visible))
		for i := start; i < end; i++ {
			sb.WriteString(renderUnusedRow(visible[i], i == m.unusedCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(visible) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(visible)-end)) + "\n")
		}
	}

	sb.WriteString("\n")
	if sel := countSelectedUnused(m.unusedPackages); sel > 0 {
		sb.WriteString(styleCheckbox.Render(fmt.Sprintf("  %d package(s) selected", sel)) + "\n")
	}
	if m.statusMsg != "" {
		sb.WriteString(styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderUnusedStatusBar(m))
	return sb.String()
}

func renderUnusedColumnHeaders() string {
	cols := fmt.Sprintf("  %-3s %-36s %-8s %-6s %s",
		"", "Package", "Type", "Source", "Dir")
	return styleMuted.Render(cols)
}

func renderUnusedRow(p *pkg.UnusedPackage, isCursor bool) string {
	check := "[ ]"
	if p.Selected {
		check = styleCheckbox.Render("[x]")
	}
	name := truncate(p.Name, 36)
	depType := styleDepType.Render(padRight(string(p.DepType), 8))
	source := padRight(p.Source, 6)
	dir := styleDirTag.Render("(" + p.DirName + ")")

	row := fmt.Sprintf("  %s %-36s %s %s %s", check, name, depType, source, dir)
	if isCursor {
		return styleCursor.Render(row)
	}
	if p.Selected {
		return styleSelected.Render(row)
	}
	return row
}

func renderUnusedStatusBar(m *Model) string {
	hints := []string{
		"j/k: navigate",
		"space: select",
		"a: all",
		"r: remove",
		"q: quit",
	}
	bar := strings.Join(hints, "  ")
	available := m.width - 2
	if available < 0 {
		available = 80
	}
	if lipgloss.Width(bar) > available {
		bar = "j/k:nav  space:sel  r:remove  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(bar)
}

func renderRemoveSummary(removed []*pkg.UnusedPackage, output string) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("\n  Removal Complete") + "\n\n")
	sb.WriteString(styleEligibilityOK.Render(fmt.Sprintf("  Removed %d package(s):", len(removed))) + "\n")
	for _, p := range removed {
		sb.WriteString(styleMuted.Render(fmt.Sprintf("    • %s (%s)", p.Name, p.Source)) + "\n")
	}
	if trimmed := strings.TrimSpace(output); trimmed != "" {
		sb.WriteString("\n" + styleMuted.Render("  Output:") + "\n")
		for _, line := range strings.Split(trimmed, "\n") {
			sb.WriteString(styleMuted.Render("    "+line) + "\n")
		}
	}
	sb.WriteString("\n" + styleMuted.Render("  Press any key to quit.") + "\n")
	return sb.String()
}

func countSelectedUnused(packages []*pkg.UnusedPackage) int {
	n := 0
	for _, p := range packages {
		if p.Selected {
			n++
		}
	}
	return n
}

func (m *Model) unusedListWindow(total int) (start, end int) {
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.unusedScroll
	if m.unusedCursor < start {
		start = m.unusedCursor
	}
	if m.unusedCursor >= start+maxVisible {
		start = m.unusedCursor - maxVisible + 1
	}
	m.unusedScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}

func pluralIes(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
