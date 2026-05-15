package depsort

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/marcelmettler/chorekit/internal/pkg"
)

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

var depSections = []string{
	"dependencies",
	"devDependencies",
	"peerDependencies",
	"optionalDependencies",
}

// Scan walks root for package.json files and returns those with unsorted dependency sections.
func Scan(root string) ([]*pkg.SortableFile, error) {
	var result []*pkg.SortableFile

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "package.json" {
			return nil
		}
		dir := filepath.Dir(path)
		rel, _ := filepath.Rel(root, path)

		unsorted, err := findUnsortedSections(path)
		if err != nil || len(unsorted) == 0 {
			return nil
		}
		result = append(result, &pkg.SortableFile{
			File:     rel,
			Dir:      dir,
			DirName:  dirLabel(dir, root),
			Sections: unsorted,
			Selected: true,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].File < result[j].File
	})
	return result, nil
}

// Sort writes alphabetically sorted dependency sections back into the selected package.json files.
// Returns the number of files successfully sorted.
func Sort(root string, files []*pkg.SortableFile) (int, error) {
	count := 0
	for _, f := range files {
		absPath := filepath.Join(root, f.File)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return count, fmt.Errorf("%s: %w", f.File, err)
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return count, fmt.Errorf("%s: %w", f.File, err)
		}

		// Process sections in reverse order of appearance to avoid offset drift
		// when replacing bytes. Since we replace by content (not offset), order
		// doesn't matter — each section's raw bytes are distinct.
		for _, sec := range f.Sections {
			rawSection, ok := raw[sec]
			if !ok {
				continue
			}
			sorted, err := buildSortedSection(rawSection)
			if err != nil {
				continue
			}
			data = bytes.Replace(data, rawSection, sorted, 1)
		}

		if err := os.WriteFile(absPath, data, 0644); err != nil {
			return count, fmt.Errorf("%s: %w", f.File, err)
		}
		count++
	}
	return count, nil
}

func findUnsortedSections(absPath string) ([]string, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var unsorted []string
	for _, sec := range depSections {
		rawSection, ok := raw[sec]
		if !ok {
			continue
		}
		keys, err := keysInOrder(rawSection)
		if err != nil || len(keys) <= 1 {
			continue
		}
		if !sort.StringsAreSorted(keys) {
			unsorted = append(unsorted, sec)
		}
	}
	return unsorted, nil
}

// keysInOrder returns the JSON object keys in document order using a streaming decoder.
func keysInOrder(raw json.RawMessage) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected {")
	}
	var keys []string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key")
		}
		keys = append(keys, key)
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// buildSortedSection reconstructs a dependency object with keys sorted alphabetically,
// preserving the original indentation style.
func buildSortedSection(raw json.RawMessage) ([]byte, error) {
	var deps map[string]string
	if err := json.Unmarshal(raw, &deps); err != nil {
		return nil, err
	}
	if len(deps) == 0 {
		return raw, nil
	}

	keys := make([]string, 0, len(deps))
	for k := range deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entryIndent, closingIndent := detectIndents(raw)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		keyBytes, _ := json.Marshal(k)
		valBytes, _ := json.Marshal(deps[k])
		buf.WriteString("\n" + entryIndent)
		buf.Write(keyBytes)
		buf.WriteString(": ")
		buf.Write(valBytes)
		if i < len(keys)-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteString("\n" + closingIndent + "}")
	return buf.Bytes(), nil
}

// detectIndents infers the per-entry and closing-brace indentation from raw section bytes.
func detectIndents(raw json.RawMessage) (entryIndent, closingIndent string) {
	lines := bytes.Split(raw, []byte("\n"))

	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}
		spaces := 0
		for _, b := range line {
			if b == ' ' {
				spaces++
			} else {
				break
			}
		}
		if spaces > 0 {
			entryIndent = strings.Repeat(" ", spaces)
			break
		}
	}

	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimRight(lines[i], "\r")
		if len(line) == 0 {
			continue
		}
		if line[len(line)-1] == '}' {
			spaces := 0
			for _, b := range line {
				if b == ' ' {
					spaces++
				} else {
					break
				}
			}
			closingIndent = strings.Repeat(" ", spaces)
			break
		}
	}

	if entryIndent == "" {
		entryIndent = "    "
	}
	if closingIndent == "" {
		closingIndent = "  "
	}
	return
}

func dirLabel(dir, root string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return "root"
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	return parts[len(parts)-1]
}
