package changelog

import (
	"github.com/maece/bumpit/internal/pkg"
	"github.com/maece/bumpit/internal/registry"
)

// Result holds the result of a changelog fetch operation.
type Result struct {
	PackageName string
	Markdown    string
	HasBreaking bool
	Summary     string
	Highlights  pkg.Highlights
	Err         error
}

// Fetch retrieves the changelog for a package update, extracts highlights,
// and builds a composite one-line summary suitable for the list view.
func Fetch(p *pkg.PackageUpdate) Result {
	result := Result{PackageName: p.Name}

	if p.RepositoryURL == "" {
		result.Markdown = "_No changelog URL found._"
		result.Summary = "No changelog URL"
		return result
	}

	owner, repo, ok := registry.ExtractGitHubOwnerRepo(p.RepositoryURL)
	if !ok {
		result.Markdown = "_Repository is not hosted on GitHub — cannot fetch changelog automatically._"
		result.Summary = "Non-GitHub repository"
		return result
	}

	md, releaseCount, hasBreaking, err := FetchGitHubChangelog(owner, repo, p.Current, p.Latest)
	if err != nil {
		result.Err = err
		result.Markdown = "_Failed to fetch changelog: " + err.Error() + "_"
		result.Summary = "Error fetching changelog"
		return result
	}

	highlights := ExtractHighlights(md)

	result.Markdown = md
	result.HasBreaking = hasBreaking
	result.Highlights = highlights
	result.Summary = BuildSummary(highlights, releaseCount)
	return result
}
