package pnpm

import (
	"testing"

	"github.com/marcelmettler/chorekit/internal/pkg"
)

func TestClassifyLicense(t *testing.T) {
	cases := []struct {
		in   string
		want pkg.LicenseCategory
	}{
		// Plain identifiers
		{"MIT", pkg.LicenseCategoryPermissive},
		{"ISC", pkg.LicenseCategoryPermissive},
		{"Apache-2.0", pkg.LicenseCategoryPermissive},
		{"BSD-3-Clause", pkg.LicenseCategoryPermissive},
		{"OFL-1.1", pkg.LicenseCategoryPermissive},
		{"OFL-1.1-RFN", pkg.LicenseCategoryPermissive},
		{"CC-BY-4.0", pkg.LicenseCategoryPermissive},
		{"LGPL-2.1", pkg.LicenseCategoryWeakCopyleft},
		{"LGPL-3.0", pkg.LicenseCategoryWeakCopyleft},
		{"MPL-2.0", pkg.LicenseCategoryWeakCopyleft},
		{"GPL-2.0", pkg.LicenseCategoryStrongCopyleft},
		{"GPL-3.0", pkg.LicenseCategoryStrongCopyleft},
		{"AGPL-3.0", pkg.LicenseCategoryStrongCopyleft},
		{"", pkg.LicenseCategoryUnknown},
		{"SEE LICENSE IN LICENSE", pkg.LicenseCategoryUnknown},

		// -only / -or-later suffixes
		{"MIT-0", pkg.LicenseCategoryPermissive},
		{"GPL-3.0-only", pkg.LicenseCategoryStrongCopyleft},
		{"GPL-3.0-or-later", pkg.LicenseCategoryStrongCopyleft},
		{"LGPL-2.1-only", pkg.LicenseCategoryWeakCopyleft},
		{"LGPL-2.1-or-later", pkg.LicenseCategoryWeakCopyleft},

		// Legacy + suffix
		{"GPL-2.0+", pkg.LicenseCategoryStrongCopyleft},
		{"LGPL-2.1+", pkg.LicenseCategoryWeakCopyleft},

		// WITH exception — base license determines category
		{"GPL-2.0-only WITH Classpath-exception-2.0", pkg.LicenseCategoryStrongCopyleft},
		{"MIT WITH something", pkg.LicenseCategoryPermissive},

		// OR expressions — most permissive wins (consumer's choice)
		{"(MIT OR GPL-3.0-or-later)", pkg.LicenseCategoryPermissive},
		{"MIT OR GPL-3.0", pkg.LicenseCategoryPermissive},
		{"MIT OR Apache-2.0", pkg.LicenseCategoryPermissive},
		{"LGPL-2.1 OR GPL-3.0", pkg.LicenseCategoryWeakCopyleft},
		{"GPL-2.0 OR GPL-3.0", pkg.LicenseCategoryStrongCopyleft},

		// AND expressions — most restrictive wins
		{"MIT AND Apache-2.0", pkg.LicenseCategoryPermissive},
		{"MIT AND LGPL-2.1", pkg.LicenseCategoryWeakCopyleft},
		{"MIT AND GPL-3.0", pkg.LicenseCategoryStrongCopyleft},
		{"LGPL-2.1 AND GPL-3.0", pkg.LicenseCategoryStrongCopyleft},

		// Nested / complex
		{"(MIT OR Apache-2.0) AND LGPL-2.1", pkg.LicenseCategoryWeakCopyleft},
		{"(GPL-3.0 OR MIT) AND (Apache-2.0 OR ISC)", pkg.LicenseCategoryPermissive},
	}

	for _, c := range cases {
		got := classifyLicense(c.in)
		if got != c.want {
			t.Errorf("classifyLicense(%q) = %v (%s), want %v (%s)",
				c.in, got, got.Label(), c.want, c.want.Label())
		}
	}
}
