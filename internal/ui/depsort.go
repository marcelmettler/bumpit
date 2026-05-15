package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcelmettler/chorekit/internal/pkg"
)

func renderSortList(m *Model) string {
	var sb strings.Builder

	count := len(m.sortFiles)
	sb.WriteString(styleTitle.Render("  Sort Dependencies") + "\n")
	if count == 0 {
		sb.WriteString(styleEligibilityOK.Render("  All dependency sections are already sorted alphabetically.") + "\n")
		sb.WriteString(renderSortStatusBar(m))
		return sb.String()
	}
	sb.WriteString(styleMuted.Render(fmt.Sprintf(
		"  %d file%s with unsorted dependency sections",
		count, pluralS(count),
	)) + "\n\n")

	sb.WriteString(renderSortColumnHeaders() + "\n")

	start, end := m.sortListWindow(len(m.sortFiles))
	for i := start; i < end; i++ {
		sb.WriteString(renderSortRow(m.sortFiles[i], i == m.sortCursor) + "\n")
	}
	if start > 0 {
		sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
	}
	if end < len(m.sortFiles) {
		sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(m.sortFiles)-end)) + "\n")
	}

	sb.WriteString("\n")
	if sel := countSelectedSort(m.sortFiles); sel > 0 {
		sb.WriteString(styleCheckbox.Render(fmt.Sprintf("  %d file(s) selected", sel)) + "\n")
	}
	if m.statusMsg != "" {
		sb.WriteString(styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderSortStatusBar(m))
	return sb.String()
}

func renderSortColumnHeaders() string {
	cols := fmt.Sprintf("  %-3s %-44s %-30s %s",
		"", "File", "Unsorted Sections", "Dir")
	return styleMuted.Render(cols)
}

func renderSortRow(f *pkg.SortableFile, isCursor bool) string {
	check := "[ ]"
	if f.Selected {
		check = styleCheckbox.Render("[x]")
	}
	file := truncate(f.File, 44)
	sections := styleWarning.Render(truncate(strings.Join(f.Sections, ", "), 30))
	dir := styleDirTag.Render("(" + f.DirName + ")")

	row := fmt.Sprintf("  %s %-44s %-30s %s", check, file, sections, dir)
	if isCursor {
		return styleCursor.Render(row)
	}
	if f.Selected {
		return styleSelected.Render(row)
	}
	return row
}

func renderSortStatusBar(m *Model) string {
	hints := []string{
		"j/k: navigate",
		"space: select",
		"a: all",
		"s: sort selected",
		"q: quit",
	}
	bar := strings.Join(hints, "  ")
	available := m.width - 2
	if available < 0 {
		available = 80
	}
	if lipgloss.Width(bar) > available {
		bar = "j/k:nav  space:sel  s:sort  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(bar)
}

func renderSortDone(count int, sorted []*pkg.SortableFile) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("\n  Sort Complete") + "\n\n")
	sb.WriteString(styleEligibilityOK.Render(fmt.Sprintf("  Sorted %d file%s:", count, pluralS(count))) + "\n")

	for _, f := range sorted {
		sb.WriteString(styleMuted.Render(fmt.Sprintf("\n  %s", f.File)) + "\n")
		for _, sec := range f.Sections {
			sb.WriteString(styleMuted.Render(fmt.Sprintf(
				"    • %s", styleEligibilityOK.Render(sec),
			)) + "\n")
		}
	}
	sb.WriteString("\n" + styleMuted.Render("  Press any key to quit.") + "\n")
	return sb.String()
}

func countSelectedSort(files []*pkg.SortableFile) int {
	n := 0
	for _, f := range files {
		if f.Selected {
			n++
		}
	}
	return n
}

func (m *Model) sortListWindow(total int) (start, end int) {
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.sortScroll
	if m.sortCursor < start {
		start = m.sortCursor
	}
	if m.sortCursor >= start+maxVisible {
		start = m.sortCursor - maxVisible + 1
	}
	m.sortScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}
