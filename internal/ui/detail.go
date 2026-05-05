package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/maece/bumpit/internal/pkg"
)

// renderDetail renders the detail view for a single package.
//
// Layout:
//   fixed  │ blank + name/version line + meta line  (headerLineCount lines)
//   scroll │ banners + highlights + changelog        (pageSize lines)
//   fixed  │ scroll indicator                        (1 line)
//   fixed  │ status bar                              (1 line)
func renderDetail(m *Model) string {
	if m.detailPackage == nil {
		return "No package selected."
	}

	p := m.detailPackage

	// ── Fixed header (always visible) ────────────────────────────────────────
	var header strings.Builder
	header.WriteString("\n")
	header.WriteString(fmt.Sprintf("%s  %s → %s  (%s)\n",
		styleHeader.Render(p.Name),
		p.Current,
		styleMajor.Render(p.Latest),
		renderKind(p.Kind),
	))
	header.WriteString(styleMuted.Render(fmt.Sprintf(
		"  Published: %s ago  |  Dep type: %s  |  Dir: %s",
		p.AgeDisplay(), string(p.DepType), p.DirName,
	)) + "\n")
	headerStr := header.String()
	headerLines := strings.Count(headerStr, "\n") // blank + name + meta = 3

	// ── Scrollable body ───────────────────────────────────────────────────────
	var body strings.Builder

	if p.VulnCount > 0 {
		body.WriteString(styleBreakingBanner.Render(
			fmt.Sprintf("  ⚠ %d known vulnerabilities", p.VulnCount)) + "\n")
	}
	if p.HasBreaking {
		body.WriteString(styleBreakingBanner.Render(
			"  ⚠  BREAKING CHANGES DETECTED — review carefully before updating  ⚠  ") + "\n")
	}
	if !p.IsEligible && !p.EligibleAt.IsZero() {
		body.WriteString(styleWarning.Render(
			fmt.Sprintf("  ⏳ Not yet eligible — eligible after %s", p.EligibleAt.Format("2006-01-02"))) + "\n")
	}

	if !p.ChangelogFetched {
		body.WriteString("\n" + styleMuted.Render("  Loading changelog...") + "\n")
	} else if p.Changelog == "" {
		body.WriteString("\n" + styleMuted.Render("  No changelog available.") + "\n")
	} else {
		body.WriteString("\n")
		if !p.Highlights.IsEmpty() {
			body.WriteString(renderHighlights(p.Highlights, m.width) + "\n")
		}
		body.WriteString(styleHighlightHeader.Render("  Full Changelog") + "\n")

		rendered := p.CachedRender(m.width)
		if rendered == "" {
			rendered = renderChangelogHighlighted(p.Changelog, m.width)
			p.SetCachedRender(rendered, m.width)
		}
		body.WriteString(rendered)
	}

	// Split body into lines; strip trailing empty lines produced by glamour.
	bodyLines := strings.Split(body.String(), "\n")
	for len(bodyLines) > 1 && bodyLines[len(bodyLines)-1] == "" {
		bodyLines = bodyLines[:len(bodyLines)-1]
	}

	// ── Page size: height = headerLines + pageSize + 1 (indicator) + 1 (bar) ─
	pageSize := m.height - headerLines - 2
	if pageSize < 5 {
		pageSize = 5
	}

	maxStart := len(bodyLines) - pageSize
	if maxStart < 0 {
		maxStart = 0
	}
	start := m.detailScroll
	if start > maxStart {
		start = maxStart
	}
	end := start + pageSize
	if end > len(bodyLines) {
		end = len(bodyLines)
	}

	// ── Assemble ──────────────────────────────────────────────────────────────
	var sb strings.Builder
	sb.WriteString(headerStr)
	sb.WriteString(strings.Join(bodyLines[start:end], "\n"))

	// Scroll indicator (always reserve 1 line so status bar stays pinned).
	if len(bodyLines) > pageSize {
		pct := 0
		if len(bodyLines) > 0 {
			pct = (start * 100) / len(bodyLines)
		}
		sb.WriteString("\n" + styleScrollIndicator.Render(
			fmt.Sprintf("  %d–%d / %d  (%d%%)  j/k: scroll",
				start+1, end, len(bodyLines), pct),
		))
	} else {
		sb.WriteString("\n")
	}

	// Status bar
	selLabel := "[ ] not selected"
	if p.Selected {
		selLabel = "[x] selected"
	}
	sb.WriteString("\n" + styleStatusBar.Width(m.width).Render(
		fmt.Sprintf("esc: back  space: %s  u: update this  j/k: scroll", selLabel),
	))

	return sb.String()
}

// renderHighlights renders the highlights panel shown above the full changelog.
func renderHighlights(h pkg.Highlights, width int) string {
	var sb strings.Builder

	writeSection := func(label, icon string, items []string, limit int) {
		if len(items) == 0 {
			return
		}
		sb.WriteString(styleHighlightHeader.Render(icon+" "+label) + "\n")
		shown := items
		if len(shown) > limit {
			shown = shown[:limit]
		}
		for _, item := range shown {
			sb.WriteString(styleMuted.Render("  • "+truncateItem(item, width-8)) + "\n")
		}
		if len(items) > limit {
			sb.WriteString(styleMuted.Render(fmt.Sprintf("  … and %d more", len(items)-limit)) + "\n")
		}
		sb.WriteString("\n")
	}

	writeSection("Breaking Changes", "⚠", h.Breaking, 5)
	writeSection("New Features", "✦", h.Features, 5)
	writeSection("Bug Fixes", "✓", h.Fixes, 3)
	if len(h.Migration) > 0 {
		writeSection("Migration Guide", "→", h.Migration, 5)
	}

	boxWidth := width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}
	return styleHighlightBox.Width(boxWidth).Render(strings.TrimRight(sb.String(), "\n"))
}

func truncateItem(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// ── Breaking-change highlighting ──────────────────────────────────────────────

// renderChangelogHighlighted renders the changelog markdown with:
//   - breaking-change sections/lines marked with a red ▌ gutter
//   - markdown links and bare URLs wrapped in OSC 8 terminal hyperlinks
func renderChangelogHighlighted(md string, width int) string {
	sections := splitChangelogSections(md)
	render := func(raw string, breaking bool) string {
		r := renderMarkdown(injectMarkdownLinks(raw), width)
		r = makeRenderedURLsClickable(r)
		return applyBreakingGutter(r, breaking)
	}
	if len(sections) == 1 {
		return render(md, false)
	}
	var out strings.Builder
	for _, s := range sections {
		out.WriteString(render(s.raw, s.breaking))
	}
	return out.String()
}

type changelogSection struct {
	raw      string
	breaking bool
}

// splitChangelogSections splits markdown at each heading boundary and marks
// sections whose header contains a breaking-change keyword.
func splitChangelogSections(md string) []changelogSection {
	var sections []changelogSection
	var cur strings.Builder
	curBreaking := false

	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			if raw := cur.String(); raw != "" {
				sections = append(sections, changelogSection{raw: raw, breaking: curBreaking})
				cur.Reset()
			}
			text := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
			curBreaking = isBreakingSectionHeader(text)
		}
		cur.WriteString(line + "\n")
	}
	if raw := cur.String(); raw != "" {
		sections = append(sections, changelogSection{raw: raw, breaking: curBreaking})
	}
	return sections
}

func isBreakingSectionHeader(text string) bool {
	for _, kw := range []string{"breaking", "incompatible"} {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// isBreakingLineContent returns true when a single rendered line (ANSI stripped,
// trimmed) carries an inline breaking-change annotation. Matches:
//   - ⚠ / ⚠️ anywhere in the line (warning emoji)
//   - "breaking change" phrase (case-insensitive)
//   - line starts with "breaking:" after stripping list markers
func isBreakingLineContent(plain string) bool {
	if strings.Contains(plain, "⚠") {
		return true
	}
	lower := strings.ToLower(plain)
	if strings.Contains(lower, "breaking change") {
		return true
	}
	// Strip common list markers then check for "breaking:" prefix
	stripped := strings.TrimLeft(plain, "•-*›>· ")
	stripped = strings.ToLower(strings.TrimSpace(stripped))
	return strings.HasPrefix(stripped, "breaking:")
}

// ── OSC 8 terminal hyperlinks ─────────────────────────────────────────────────

var (
	// matches [text](https://...) markdown link syntax
	mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\((https?://[^)\s]+)[^)]*\)`)

	// matches bare https URLs in already-rendered (ANSI) text;
	// stops at whitespace, ESC, and chars that end URLs in prose
	renderedURLRe = regexp.MustCompile(`https?://[^\s\x1b)>\]"]+`)

	ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

// osc8Link wraps text in an OSC 8 terminal hyperlink sequence.
// Terminals that don't support OSC 8 silently ignore the sequences.
func osc8Link(text, url string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// injectMarkdownLinks converts [text](url) markdown links to OSC 8 hyperlinks
// before the markdown is processed by glamour, so the URL is preserved even
// when glamour's style would otherwise hide it.
func injectMarkdownLinks(md string) string {
	return mdLinkRe.ReplaceAllStringFunc(md, func(match string) string {
		groups := mdLinkRe.FindStringSubmatch(match)
		if len(groups) < 3 {
			return match
		}
		return osc8Link(groups[1], groups[2])
	})
}

// makeRenderedURLsClickable wraps bare https:// URLs in the ANSI-rendered output
// with OSC 8 sequences. URLs that are already inside an OSC 8 sequence
// (preceded by ";;") are skipped to prevent double-wrapping.
func makeRenderedURLsClickable(s string) string {
	matches := renderedURLRe.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return s
	}
	var out strings.Builder
	out.Grow(len(s) + len(matches)*48)
	prev := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		out.WriteString(s[prev:start])
		if start >= 2 && s[start-2:start] == ";;" {
			// already inside an OSC 8 sequence — write as-is
			out.WriteString(s[start:end])
		} else {
			url := s[start:end]
			out.WriteString(osc8Link(url, url))
		}
		prev = end
	}
	out.WriteString(s[prev:])
	return out.String()
}

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// applyBreakingGutter prepends a styled red ▌ gutter marker to breaking lines.
// When sectionBreaking is true every non-blank line gets the marker (the entire
// section is a breaking-changes section). When false, only individual lines
// that carry an inline breaking-change annotation are marked.
func applyBreakingGutter(rendered string, sectionBreaking bool) string {
	marker := styleBreakingMarker.Render("▌") + " "
	lines := strings.Split(rendered, "\n")
	var out strings.Builder
	for _, line := range lines {
		plain := strings.TrimSpace(stripANSI(line))
		if plain == "" {
			out.WriteString(line + "\n")
		} else if sectionBreaking || isBreakingLineContent(plain) {
			out.WriteString(marker + line + "\n")
		} else {
			out.WriteString(line + "\n")
		}
	}
	return out.String()
}

// renderMarkdown renders markdown content using glamour.
func renderMarkdown(md string, width int) string {
	if width <= 0 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return out
}

// renderUpdateSummary renders the done state summary.
func renderUpdateSummary(updated []*pkg.PackageUpdate, output string) string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render("\n  Update Complete") + "\n\n")

	if len(updated) > 0 {
		sb.WriteString(styleMuted.Render("  Updated packages:") + "\n")
		for _, p := range updated {
			sb.WriteString(fmt.Sprintf("    %s  %s → %s\n",
				styleCheckbox.Render("✓"),
				p.Name,
				renderKind(p.Kind),
			))
		}
	}

	if output != "" {
		sb.WriteString("\n" + styleMuted.Render("  Output:") + "\n")
		for _, line := range strings.Split(output, "\n") {
			sb.WriteString("    " + line + "\n")
		}
	}

	sb.WriteString("\n" + styleStatusBar.Render("  q: quit  r: restart"))
	return sb.String()
}
