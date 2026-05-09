package ui

import (
	"fmt"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
	"github.com/marcelmettler/bumpit/internal/pkg/clean"
)

func renderCleanList(m *Model) string {
	var sb strings.Builder

	total := len(m.cleanArtifacts)
	totalSize := sumArtifactSize(m.cleanArtifacts)
	sel := selectedArtifacts(m.cleanArtifacts)
	selSize := sumArtifactSize(sel)

	sb.WriteString(styleTitle.Render("  Clean Workspace") + "\n")
	sb.WriteString(styleMuted.Render(fmt.Sprintf(
		"  %d director%s  ·  %s on disk",
		total, pluralIes(total), clean.FormatSize(totalSize),
	)) + "\n\n")

	sb.WriteString(renderCleanColumnHeaders())
	sb.WriteString("\n")

	visible := m.cleanArtifacts
	if len(visible) == 0 {
		sb.WriteString(styleEligibilityOK.Render("  Nothing to clean — workspace is already tidy.") + "\n")
	} else {
		start, end := m.cleanListWindow(len(visible))
		for i := start; i < end; i++ {
			sb.WriteString(renderCleanRow(visible[i], i == m.cleanCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(visible) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(visible)-end)) + "\n")
		}
	}

	sb.WriteString("\n")
	if len(sel) > 0 {
		sb.WriteString(styleCheckbox.Render(fmt.Sprintf(
			"  %d selected  ·  %s will be freed",
			len(sel), clean.FormatSize(selSize),
		)) + "\n")
	}
	if m.statusMsg != "" {
		sb.WriteString(styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderCleanStatusBar(m))
	return sb.String()
}

func renderCleanColumnHeaders() string {
	cols := fmt.Sprintf("  %-3s %-42s %-16s %s", "", "Path", "Kind", "Size")
	return styleMuted.Render(cols)
}

func renderCleanRow(a *pkg.ArtifactDir, isCursor bool) string {
	check := "[ ]"
	if a.Selected {
		check = styleCheckbox.Render("[x]")
	}
	path := truncate(a.RelPath, 42)
	kind := styleDirTag.Render(padRight(a.Kind, 16))
	size := styleHeader.Render(clean.FormatSize(a.Size))

	row := fmt.Sprintf("  %s %-42s %s %s", check, path, kind, size)
	if isCursor {
		return styleCursor.Render(row)
	}
	if a.Selected {
		return styleSelected.Render(row)
	}
	return row
}

func renderCleanStatusBar(m *Model) string {
	hints := []string{
		"j/k: navigate",
		"space: select",
		"a: all",
		"D: delete selected",
		"q: quit",
	}
	bar := strings.Join(hints, "  ")
	available := m.width - 2
	if available < 0 {
		available = 80
	}
	if len(bar) > available {
		bar = "j/k:nav  space:sel  D:delete  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(bar)
}

func renderCleanDone(freed int64, removed []*pkg.ArtifactDir) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("\n  Clean Complete") + "\n\n")
	sb.WriteString(styleEligibilityOK.Render(fmt.Sprintf(
		"  Freed %s across %d director%s:",
		clean.FormatSize(freed), len(removed), pluralIes(len(removed)),
	)) + "\n")
	for _, a := range removed {
		sb.WriteString(styleMuted.Render(fmt.Sprintf(
			"    • %s  (%s)",
			a.RelPath, clean.FormatSize(a.Size),
		)) + "\n")
	}
	sb.WriteString("\n" + styleMuted.Render("  Press any key to quit.") + "\n")
	return sb.String()
}

func (m *Model) cleanListWindow(total int) (start, end int) {
	maxVisible := m.height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.cleanScroll
	if m.cleanCursor < start {
		start = m.cleanCursor
	}
	if m.cleanCursor >= start+maxVisible {
		start = m.cleanCursor - maxVisible + 1
	}
	m.cleanScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}

func selectedArtifacts(artifacts []*pkg.ArtifactDir) []*pkg.ArtifactDir {
	var sel []*pkg.ArtifactDir
	for _, a := range artifacts {
		if a.Selected {
			sel = append(sel, a)
		}
	}
	return sel
}

func sumArtifactSize(artifacts []*pkg.ArtifactDir) int64 {
	var total int64
	for _, a := range artifacts {
		total += a.Size
	}
	return total
}
