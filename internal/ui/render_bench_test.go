package ui

import (
	"strings"
	"testing"
)

var benchChangelog = func() string {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("## v19.")
		sb.WriteString(strings.Repeat("x", 3))
		sb.WriteString("\n\n### Breaking Changes\n\n")
		for j := 0; j < 5; j++ {
			sb.WriteString("- Breaking change with `code` and **bold** text\n")
		}
		sb.WriteString("\n### Features\n\n")
		for j := 0; j < 10; j++ {
			sb.WriteString("- New feature added with `code` support and details\n")
		}
		sb.WriteString("\n---\n\n")
	}
	return sb.String()
}()

func BenchmarkRenderMarkdown(b *testing.B) {
	for i := 0; i < b.N; i++ {
		renderMarkdown(benchChangelog, 120)
	}
}
