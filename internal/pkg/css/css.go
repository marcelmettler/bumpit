package css

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

// ScanResult holds the outcome of a CSS unused-class scan.
type ScanResult struct {
	Unused        []*pkg.CSSClass // defined in CSS, never referenced in templates
	Undefined     []*pkg.CSSClass // referenced in templates (explicit class attrs), not defined in any CSS file
	TotalDefined  int
	TailwindFound bool
	CSSFileCount  int
	SrcFileCount  int
}

var skipDirs = map[string]bool{
	"node_modules":     true,
	".git":             true,
	"dist":             true,
	"build":            true,
	"out":              true,
	".next":            true,
	".nuxt":            true,
	".output":          true,
	".angular":         true,
	".svelte-kit":      true,
	".vite":            true,
	".turbo":           true,
	".cache":           true,
	"coverage":         true,
	".nyc_output":      true,
	"storybook-static": true,
}

var cssExts = map[string]bool{".css": true, ".scss": true, ".sass": true, ".less": true}
var srcExts = map[string]bool{
	".html": true, ".jsx": true, ".tsx": true, ".js": true,
	".ts": true, ".vue": true, ".svelte": true, ".mjs": true, ".cjs": true,
}

var tailwindConfigNames = []string{
	"tailwind.config.js", "tailwind.config.ts",
	"tailwind.config.cjs", "tailwind.config.mjs",
}

var (
	reBlockComment  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reLineComment   = regexp.MustCompile(`//[^\n]*`)
	reURL           = regexp.MustCompile(`url\([^)]*\)`)
	reCSSClass      = regexp.MustCompile(`\.([a-zA-Z][\w-]*)`)
	reClassAttr     = regexp.MustCompile(`class(?:Name)?\s*=\s*["']([^"']*)["']`)
	reStringLiteral = regexp.MustCompile(`["']([a-zA-Z][\w-]+)["']`)
)

// Scan walks root and returns which CSS classes are defined but never referenced,
// and which template class references have no corresponding CSS definition.
func Scan(root string) (*ScanResult, error) {
	// Pass 1: collect CSS class definitions.
	definedClasses := make(map[string]*pkg.CSSClass)
	cssCount := 0

	if err := walkFiles(root, func(absPath, relPath string) {
		ext := strings.ToLower(filepath.Ext(absPath))
		if !cssExts[ext] {
			return
		}
		// Skip CSS modules — class names are transformed by the bundler and
		// referenced as styles.foo rather than as string literals.
		if strings.Contains(filepath.Base(absPath), ".module.") {
			return
		}
		cssCount++
		content, err := os.ReadFile(absPath)
		if err != nil {
			return
		}
		for _, c := range extractCSSClasses(content, relPath) {
			if _, exists := definedClasses[c.Name]; !exists {
				definedClasses[c.Name] = c
			}
		}
	}); err != nil {
		return nil, err
	}

	// Pass 2: collect class references from source/template files.
	// usedBroad includes string literals — conservative, avoids false positives in "unused" direction.
	// usedExplicit tracks only explicit class= attributes with location — used for "undefined" direction.
	usedBroad := make(map[string]bool)
	usedExplicit := make(map[string]*pkg.CSSClass)
	srcCount := 0

	if err := walkFiles(root, func(absPath, relPath string) {
		ext := strings.ToLower(filepath.Ext(absPath))
		if !srcExts[ext] {
			return
		}
		srcCount++
		content, err := os.ReadFile(absPath)
		if err != nil {
			return
		}
		broad, explicit := extractClassRefs(content, relPath)
		for name := range broad {
			usedBroad[name] = true
		}
		for name, c := range explicit {
			if _, exists := usedExplicit[name]; !exists {
				usedExplicit[name] = c
			}
		}
	}); err != nil {
		return nil, err
	}

	// Unused: defined in CSS but absent from the broad usage set.
	var unused []*pkg.CSSClass
	for name, c := range definedClasses {
		if !usedBroad[name] {
			unused = append(unused, c)
		}
	}
	sort.Slice(unused, func(i, j int) bool {
		if unused[i].File != unused[j].File {
			return unused[i].File < unused[j].File
		}
		return unused[i].Line < unused[j].Line
	})

	// Undefined: explicitly referenced in templates but absent from CSS definitions.
	var undefined []*pkg.CSSClass
	for name, c := range usedExplicit {
		if _, exists := definedClasses[name]; !exists {
			undefined = append(undefined, c)
		}
	}
	sort.Slice(undefined, func(i, j int) bool {
		if undefined[i].File != undefined[j].File {
			return undefined[i].File < undefined[j].File
		}
		return undefined[i].Line < undefined[j].Line
	})

	return &ScanResult{
		Unused:        unused,
		Undefined:     undefined,
		TotalDefined:  len(definedClasses),
		TailwindFound: detectTailwind(root),
		CSSFileCount:  cssCount,
		SrcFileCount:  srcCount,
	}, nil
}

func extractCSSClasses(content []byte, relPath string) []*pkg.CSSClass {
	content = reBlockComment.ReplaceAll(content, nil)
	content = reLineComment.ReplaceAll(content, nil)
	content = reURL.ReplaceAll(content, []byte("url()"))

	var classes []*pkg.CSSClass
	seen := make(map[string]bool)

	for lineNum, line := range strings.Split(string(content), "\n") {
		// Skip SCSS parent-selector lines — &__child, &:hover etc. can't be
		// statically resolved without tracking the full nesting context.
		if strings.Contains(strings.TrimSpace(line), "&") {
			continue
		}
		for _, m := range reCSSClass.FindAllStringSubmatch(line, -1) {
			name := m[1]
			if seen[name] {
				continue
			}
			seen[name] = true
			classes = append(classes, &pkg.CSSClass{
				Name: name,
				File: relPath,
				Line: lineNum + 1,
			})
		}
	}
	return classes
}

// extractClassRefs returns two maps from a source file.
// broad includes class= attributes and string literals — used to determine "defined but unused".
// explicit includes only class= attribute values with their location — used to determine "used but undefined".
// String literals are intentionally excluded from explicit to avoid false positives from common English words.
func extractClassRefs(content []byte, relPath string) (broad map[string]bool, explicit map[string]*pkg.CSSClass) {
	broad = make(map[string]bool)
	explicit = make(map[string]*pkg.CSSClass)

	for lineNum, line := range strings.Split(string(content), "\n") {
		lb := []byte(line)
		for _, m := range reClassAttr.FindAllSubmatch(lb, -1) {
			for _, name := range strings.Fields(string(m[1])) {
				broad[name] = true
				if _, ok := explicit[name]; !ok {
					explicit[name] = &pkg.CSSClass{Name: name, File: relPath, Line: lineNum + 1}
				}
			}
		}
		// String literals go into broad only — catches cx('foo'), clsx('bar'), classList.add('baz').
		for _, m := range reStringLiteral.FindAllSubmatch(lb, -1) {
			broad[string(m[1])] = true
		}
	}
	return
}

func detectTailwind(root string) bool {
	for _, name := range tailwindConfigNames {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

func walkFiles(root string, fn func(absPath, relPath string)) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		fn(path, rel)
		return nil
	})
}
