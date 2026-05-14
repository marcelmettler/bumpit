package clean

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/marcelmettler/chorekit/internal/pkg"
)

// artifactKinds maps a directory name to its human-readable kind label.
// Directories listed here are considered safe to delete.
var artifactKinds = map[string]string{
	"node_modules":     "dependencies",
	"dist":             "build output",
	"build":            "build output",
	"out":              "build output",
	"storybook-static": "build output",
	".next":            "Next.js",
	".nuxt":            "Nuxt",
	".output":          "Nuxt output",
	".angular":         "Angular cache",
	".svelte-kit":      "SvelteKit",
	".vite":            "Vite cache",
	".turbo":           "Turbo cache",
	".cache":           "cache",
	"coverage":         "test coverage",
	".nyc_output":      "test coverage",
}

// FindArtifacts walks root and returns all artifact directories with their sizes.
// It does not descend into artifact directories (so nested node_modules are only
// reported at the outermost level).
func FindArtifacts(root string) ([]*pkg.ArtifactDir, error) {
	var artifacts []*pkg.ArtifactDir

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == root {
			return nil
		}
		if d.Name() == ".git" {
			return fs.SkipDir
		}
		kind, ok := artifactKinds[d.Name()]
		if !ok {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		size, _ := dirSize(path)
		artifacts = append(artifacts, &pkg.ArtifactDir{
			Path:    path,
			RelPath: filepath.ToSlash(rel),
			Kind:    kind,
			Size:    size,
		})
		return fs.SkipDir // don't recurse into artifact dirs
	})
	if err != nil {
		return nil, err
	}

	// Biggest directories first — most impactful to delete.
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Size > artifacts[j].Size
	})
	return artifacts, nil
}

// Remove deletes each selected artifact directory and returns the total bytes freed.
func Remove(artifacts []*pkg.ArtifactDir) (freed int64, err error) {
	for _, a := range artifacts {
		if removeErr := os.RemoveAll(a.Path); removeErr != nil {
			return freed, fmt.Errorf("remove %s: %w", a.RelPath, removeErr)
		}
		freed += a.Size
	}
	return freed, nil
}

// FormatSize formats a byte count as a human-readable string.
func FormatSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%d MB", b/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%d KB", b/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func dirSize(path string) (int64, error) {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, nil
}
