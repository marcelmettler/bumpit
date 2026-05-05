package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/maece/bumpit/internal/pkg"
)

// renderList renders the main list view.
func renderList(m *Model) string {
	var sb strings.Builder

	// Header
	title := styleTitle.Render("  Package Updater")
	countInfo := styleMuted.Render(fmt.Sprintf("  %d packages", len(m.packages)))
	if m.filterQuery != "" {
		countInfo += styleMuted.Render(fmt.Sprintf("  (filter: %q, %d shown)", m.filterQuery, len(m.filtered)))
	}
	sb.WriteString(title + "\n")
	sb.WriteString(countInfo + "\n")

	// Sort indicator
	sortLabel := styleMuted.Render(fmt.Sprintf("  Sort: %s  [s to cycle]", m.sortMode.String()))
	sb.WriteString(sortLabel + "\n\n")

	// Filter input
	if m.filterActive {
		sb.WriteString(styleHeader.Render("  Filter: ") + m.filterQuery + "█\n\n")
	}

	// Column headers
	sb.WriteString(renderColumnHeaders())
	sb.WriteString("\n")

	// List items
	visible := m.filtered
	if len(visible) == 0 {
		sb.WriteString(styleMuted.Render("  No packages match your filter.") + "\n")
	} else {
		// Show a window of items based on viewport
		start, end := m.listWindow(len(visible))
		for i := start; i < end; i++ {
			p := visible[i]
			sb.WriteString(renderListRow(p, i == m.cursor, m.width) + "\n")
			// Show composite summary inline under the cursor row
			if i == m.cursor && p.ChangelogFetched && p.OneLineSummary != "" {
				sb.WriteString(styleSummaryLine.Render("       "+p.OneLineSummary) + "\n")
			}
		}
		// Scroll indicators
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(visible) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(visible)-end)) + "\n")
		}
	}

	// Selected count
	selCount := countSelected(m.packages)
	sb.WriteString("\n")
	if selCount > 0 {
		sb.WriteString(styleCheckbox.Render(fmt.Sprintf("  %d package(s) selected", selCount)) + "\n")
	}

	// Status / help bar
	sb.WriteString(renderStatusBar(m))

	return sb.String()
}

func renderColumnHeaders() string {
	cols := fmt.Sprintf("  %-3s %-24s %-10s   %-10s %-7s %-14s %s",
		"", "Package", "Current", "Latest", "Kind", "Status", "Dir/Type")
	return styleMuted.Render(cols)
}

func renderListRow(p *pkg.PackageUpdate, isCursor bool, _ int) string {
	// Checkbox
	check := "[ ]"
	if p.Selected {
		check = styleCheckbox.Render("[x]")
	}

	// Package name (truncate if needed)
	name := truncate(p.Name, 24)
	current := padLeft(p.Current, 10)
	latest := padLeft(p.Latest, 10)

	// Kind with color
	kindStr := renderKind(p.Kind)

	// Status (breaking / eligible / OK)
	status := renderStatus(p)

	// Dir + dep type
	dir := styleDirTag.Render("(" + p.DirName + ")")
	depType := styleDepType.Render(string(p.DepType))
	meta := dir + " " + depType

	row := fmt.Sprintf("  %s %-24s %s → %s %s %s %s",
		check, name, current, latest, kindStr, status, meta)

	// Vulnerability indicator
	if p.VulnCount > 0 {
		row += styleBreakingBanner.Render(fmt.Sprintf(" %d vuln", p.VulnCount))
	}

	if isCursor {
		// Highlight the entire row
		row = styleCursor.Render(row)
	} else if p.Selected {
		row = styleSelected.Render(row)
	}

	return row
}

func renderKind(k pkg.UpdateKind) string {
	switch k {
	case pkg.KindMajor:
		return styleMajor.Render(padRight("MAJOR", 7))
	case pkg.KindMinor:
		return styleMinor.Render(padRight("minor", 7))
	case pkg.KindPatch:
		return stylePatch.Render(padRight("patch", 7))
	default:
		return styleUnknown.Render(padRight("?????", 7))
	}
}

func renderStatus(p *pkg.PackageUpdate) string {
	width := 14
	if !p.IsEligible && !p.EligibleAt.IsZero() {
		remaining := time.Until(p.EligibleAt)
		if remaining > 0 {
			days := int(remaining.Hours()/24) + 1
			s := fmt.Sprintf("⏳ %dd left", days)
			return styleEligibilityBad.Render(padRight(s, width))
		}
	}
	if !p.ChangelogFetched {
		return styleMuted.Render(padRight("…", width))
	}
	if p.HasBreaking {
		return styleBreakingBanner.Render(padRight("⚠ Breaking", width))
	}
	return styleEligibilityOK.Render(padRight("✓ OK", width))
}

func renderStatusBar(m *Model) string {
	hints := []string{
		"j/k: navigate",
		"space: select",
		"a: all",
		"enter: detail",
		"/: filter",
		"u: update",
		"s: sort",
		"?: help",
		"q: quit",
	}
	bar := strings.Join(hints, "  ")
	available := m.width - 2
	if available < 0 {
		available = 80
	}
	if lipgloss.Width(bar) > available {
		bar = "j/k:nav  space:sel  u:update  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(bar)
}

func renderHelpOverlay() string {
	content := `
Key Bindings
────────────────────────────────────
Navigation
  j / ↓       Move cursor down
  k / ↑       Move cursor up

Selection
  space       Toggle selection
  a           Select/deselect all visible

Actions
  enter       View changelog detail
  u           Update selected packages
  s           Cycle sort order (name/kind/age)
  /           Start filter
  esc         Clear filter / go back

View
  ?           Toggle this help
  q           Quit
`
	return styleHelp.Render(strings.TrimLeft(content, "\n"))
}

func countSelected(packages []*pkg.PackageUpdate) int {
	n := 0
	for _, p := range packages {
		if p.Selected {
			n++
		}
	}
	return n
}

// listWindow returns the visible slice [start, end) for scrolling.
func (m *Model) listWindow(total int) (start, end int) {
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	// Keep cursor in view
	start = m.scrollOffset
	if m.cursor < start {
		start = m.cursor
	}
	if m.cursor >= start+maxVisible {
		start = m.cursor - maxVisible + 1
	}
	m.scrollOffset = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return padRight(s, max)
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}

func padRight(s string, n int) string {
	l := utf8.RuneCountInString(s)
	if l >= n {
		return s
	}
	return s + strings.Repeat(" ", n-l)
}

func padLeft(s string, n int) string {
	l := utf8.RuneCountInString(s)
	if l >= n {
		return s
	}
	return strings.Repeat(" ", n-l) + s
}
