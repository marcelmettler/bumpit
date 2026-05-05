package auth

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GitHubToken resolves the best available GitHub token without requiring the
// user to configure anything manually. Resolution order:
//
//  1. GITHUB_TOKEN env var  (explicit override)
//  2. GH_TOKEN env var       (GitHub CLI convention)
//  3. gh auth token          (GitHub CLI credential store)
//  4. git credential fill    (system store: macOS Keychain, libsecret, etc.)
//  5. empty string           (unauthenticated, 60 req/hr)
func GitHubToken() string {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t
	}
	if t := ghCLIToken(); t != "" {
		return t
	}
	if t := gitCredentialToken(); t != "" {
		return t
	}
	return ""
}

// ghCLIToken asks the GitHub CLI for the current token.
// Returns "" if gh is not installed or the user is not logged in.
func ghCLIToken() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// gitCredentialToken queries the system credential helper for github.com.
// On macOS this reads from the Keychain (via git-credential-osxkeychain).
// On Linux it uses git-credential-libsecret or similar.
// The stored password is typically an OAuth token, not an actual password,
// when the user set up git with HTTPS and authenticated via GitHub.
func gitCredentialToken() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n\n")

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		if after, ok := strings.CutPrefix(line, "password="); ok {
			if t := strings.TrimSpace(after); t != "" {
				return t
			}
		}
	}
	return ""
}
