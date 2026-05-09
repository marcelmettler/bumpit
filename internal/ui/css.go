package ui

import (
	"fmt"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

// cssItem is a row in the combined CSS audit list.
type cssItem struct {
	class     *pkg.CSSClass
	undefined bool // true = referenced in template but missing from CSS; false = defined but unused
}

func renderCSSList(m *Model) string {
	var sb strings.Builder

	result := m.cssResult
	if result == nil {
		return ""
	}

	sb.WriteString(styleTitle.Render("  CSS Audit") + "\n")
	sb.WriteString(styleMuted.Render(fmt.Sprintf(
		"  %d CSS file%s  ·  %d class%s defined  ·  %d template%s scanned",
		result.CSSFileCount, pluralS(result.CSSFileCount),
		result.TotalDefined, pluralEs(result.TotalDefined),
		result.SrcFileCount, pluralS(result.SrcFileCount),
	)) + "\n")

	if result.TailwindFound {
		sb.WriteString(styleWarning.Render("  ⚠  Tailwind detected — utility classes are not tracked by this scan") + "\n")
	}

	nUnused := len(result.Unused)
	nUndef := len(result.Undefined)
	if nUnused > 0 || nUndef > 0 {
		var findings []string
		if nUnused > 0 {
			findings = append(findings, styleWarning.Render(fmt.Sprintf("%d defined but unused", nUnused)))
		}
		if nUndef > 0 {
			findings = append(findings, styleMajor.Render(fmt.Sprintf("%d used but undefined", nUndef)))
		}
		sb.WriteString(styleMuted.Render("  ") + strings.Join(findings, styleMuted.Render("  ·  ")) + "\n")
	}

	sb.WriteString("\n")

	if m.cssFilterActive {
		sb.WriteString(styleHeader.Render(fmt.Sprintf("  Filter: %s_", m.cssFilterQuery)) + "\n")
	}

	if len(m.cssItems) == 0 {
		if result.TotalDefined == 0 && result.CSSFileCount == 0 {
			sb.WriteString(styleMuted.Render("  No CSS files found.") + "\n")
		} else if m.cssFilterQuery != "" {
			sb.WriteString(styleMuted.Render("  No classes match the filter.") + "\n")
		} else {
			sb.WriteString(styleEligibilityOK.Render("  All CSS classes are referenced and all template classes are defined.") + "\n")
		}
	} else {
		sb.WriteString(renderCSSColumnHeaders() + "\n")

		start, end := m.cssListWindow(len(m.cssItems))
		for i := start; i < end; i++ {
			sb.WriteString(renderCSSItem(m.cssItems[i], i == m.cssCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(m.cssItems) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(m.cssItems)-end)) + "\n")
		}
	}

	if m.statusMsg != "" {
		sb.WriteString("\n" + styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderCSSStatusBar(m))
	return sb.String()
}

func renderCSSColumnHeaders() string {
	return styleMuted.Render(fmt.Sprintf("  %-48s %-6s %-2s %s", "File", "Line", "", "Class"))
}

func renderCSSItem(item cssItem, isCursor bool) string {
	file := truncate(item.class.File, 48)
	line := fmt.Sprintf("%-6d", item.class.Line)

	var indicator, name string
	if item.undefined {
		indicator = styleMajor.Render("?")
		name = styleMajor.Render("." + item.class.Name)
	} else {
		indicator = styleWarning.Render("~")
		name = styleWarning.Render("." + item.class.Name)
	}

	row := fmt.Sprintf("  %-48s %s %s %s", file, line, indicator, name)
	if isCursor {
		return styleCursor.Render(row)
	}
	return row
}

func renderCSSStatusBar(m *Model) string {
	legend := "~:unused  ?:undefined"
	hints := legend + "   j/k: navigate  /: filter  q: quit"
	if m.width-2 < len(hints) {
		hints = "j/k:nav  /:filter  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(hints)
}

func (m *Model) cssListWindow(total int) (start, end int) {
	maxVisible := m.height - 14
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.cssScroll
	if m.cssCursor < start {
		start = m.cssCursor
	}
	if m.cssCursor >= start+maxVisible {
		start = m.cssCursor - maxVisible + 1
	}
	m.cssScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func pluralEs(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}
