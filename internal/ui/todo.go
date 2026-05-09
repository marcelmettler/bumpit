package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcelmettler/bumpit/internal/pkg"
)

var (
	styleTodoFixme = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800")).Bold(true)
	styleTodoHack  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800"))
	styleTodoXXX   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Bold(true)
	styleTodoTodo  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5599FF"))
)

func renderTodoList(m *Model) string {
	var sb strings.Builder

	result := m.todoResult
	if result == nil {
		return ""
	}

	total := len(result.Items)
	counts := countByKind(result.Items)

	sb.WriteString(styleTitle.Render("  TODO Audit") + "\n")

	if total == 0 {
		sb.WriteString(styleMuted.Render("  No TODO comments found.") + "\n")
	} else {
		summary := fmt.Sprintf("  %d comment%s", total, pluralS(total))
		var parts []string
		for _, kind := range []string{"TODO", "FIXME", "HACK", "XXX"} {
			if n := counts[kind]; n > 0 {
				parts = append(parts, todoKindStyle(kind).Render(fmt.Sprintf("%d %s", n, kind)))
			}
		}
		if len(parts) > 0 {
			summary += "  " + strings.Join(parts, styleMuted.Render("  ·  "))
		}
		sb.WriteString(styleMuted.Render(summary) + "\n")
	}

	sb.WriteString("\n")

	if m.todoFilterActive {
		sb.WriteString(styleHeader.Render(fmt.Sprintf("  Filter: %s_", m.todoFilterQuery)) + "\n")
	}

	if len(m.todoFiltered) == 0 {
		if m.todoFilterQuery != "" {
			sb.WriteString(styleMuted.Render("  No comments match the filter.") + "\n")
		} else if total > 0 {
			sb.WriteString(styleEligibilityOK.Render("  No TODO comments found.") + "\n")
		}
	} else {
		sb.WriteString(renderTodoColumnHeaders() + "\n")
		start, end := m.todoListWindow(len(m.todoFiltered))
		for i := start; i < end; i++ {
			sb.WriteString(renderTodoRow(m.todoFiltered[i], i == m.todoCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(m.todoFiltered) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(m.todoFiltered)-end)) + "\n")
		}
	}

	if m.statusMsg != "" {
		sb.WriteString("\n" + styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderTodoStatusBar(m))
	return sb.String()
}

func renderTodoColumnHeaders() string {
	return styleMuted.Render(fmt.Sprintf("  %-44s %-6s %-6s %s", "File", "Line", "Kind", "Comment"))
}

func renderTodoRow(item *pkg.TodoItem, isCursor bool) string {
	file := truncate(item.File, 44)
	line := fmt.Sprintf("%-6d", item.Line)
	kind := todoKindStyle(item.Kind).Render(fmt.Sprintf("%-6s", item.Kind))
	text := truncate(item.Text, 60)
	row := fmt.Sprintf("  %-44s %s %s %s", file, line, kind, text)
	if isCursor {
		return styleCursor.Render(row)
	}
	return row
}

func renderTodoStatusBar(m *Model) string {
	hints := "j/k: navigate  /: filter  q: quit"
	return "\n" + styleStatusBar.Width(m.width).Render(hints)
}

func (m *Model) todoListWindow(total int) (start, end int) {
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.todoScroll
	if m.todoCursor < start {
		start = m.todoCursor
	}
	if m.todoCursor >= start+maxVisible {
		start = m.todoCursor - maxVisible + 1
	}
	m.todoScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}

func todoKindStyle(kind string) lipgloss.Style {
	switch kind {
	case "FIXME":
		return styleTodoFixme
	case "HACK":
		return styleTodoHack
	case "XXX":
		return styleTodoXXX
	default:
		return styleTodoTodo
	}
}

func countByKind(items []*pkg.TodoItem) map[string]int {
	counts := make(map[string]int)
	for _, item := range items {
		counts[item.Kind]++
	}
	return counts
}
