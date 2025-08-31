package mod_test

import (
	"reflect"
	"testing"

	"github.com/concordalabs/monogo/mod"
	"golang.org/x/mod/modfile"
)

func TestDiff(t *testing.T) {
	testCases := []struct {
		name     string
		left     string
		right    string
		expected mod.Output
	}{
		{
			name: "no changes",
			left: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
)
`,
			right: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
)
`,
			expected: mod.Output{Type: mod.ChangeNone},
		},
		{
			name: "go version change",
			left: `
module my/project
go 1.20
`,
			right: `
module my/project
go 1.21
`,
			expected: mod.Output{Type: mod.ChangeGolang},
		},
		{
			name: "toolchain version change",
			left: `
module my/project
go 1.21
toolchain go1.21.0
`,
			right: `
module my/project
go 1.21
toolchain go1.22.0
`,
			expected: mod.Output{Type: mod.ChangeGolangToolchain},
		},
		{
			name: "package added",
			left: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
)
`,
			right: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
    "example.com/b" v1.0.0
)
`,
			expected: mod.Output{
				Type: mod.ChangePackages,
				Packages: mod.ChangedPackages{
					Added: []string{"example.com/b"},
					None:  []string{"example.com/a"},
				},
			},
		},
		{
			name: "package deleted",
			left: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
    "example.com/b" v1.0.0
)
`,
			right: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
)
`,
			expected: mod.Output{
				Type: mod.ChangePackages,
				Packages: mod.ChangedPackages{
					Deleted: []string{"example.com/b"},
					None:    []string{"example.com/a"},
				},
			},
		},
		{
			name: "package changed",
			left: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
)
`,
			right: `
module my/project
go 1.21
require (
    "example.com/a" v1.1.0
)
`,
			expected: mod.Output{
				Type: mod.ChangePackages,
				Packages: mod.ChangedPackages{
					Changed: []string{"example.com/a"},
				},
			},
		},
		{
			name: "multiple changes",
			left: `
module my/project
go 1.21
require (
    "example.com/a" v1.0.0
    "example.com/b" v1.0.0
)
`,
			right: `
module my/project
go 1.21
require (
    "example.com/a" v1.1.0
    "example.com/c" v1.0.0
)
`,
			expected: mod.Output{
				Type: mod.ChangePackages,
				Packages: mod.ChangedPackages{
					Added:   []string{"example.com/c"},
					Deleted: []string{"example.com/b"},
					Changed: []string{"example.com/a"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			leftMod, err := modfile.Parse("go.mod", []byte(tc.left), nil)
			if err != nil {
				t.Fatalf("failed to parse left go.mod: %s", err)
			}

			rightMod, err := modfile.Parse("go.mod", []byte(tc.right), nil)
			if err != nil {
				t.Fatalf("failed to parse right go.mod: %s", err)
			}

			output := mod.Diff(leftMod, rightMod)

			// Normalise slices for comparison
			if output.Packages.Added == nil {
				output.Packages.Added = []string{}
			}
			if output.Packages.Deleted == nil {
				output.Packages.Deleted = []string{}
			}
			if output.Packages.Changed == nil {
				output.Packages.Changed = []string{}
			}
			if output.Packages.None == nil {
				output.Packages.None = []string{}
			}
			if tc.expected.Packages.Added == nil {
				tc.expected.Packages.Added = []string{}
			}
			if tc.expected.Packages.Deleted == nil {
				tc.expected.Packages.Deleted = []string{}
			}
			if tc.expected.Packages.Changed == nil {
				tc.expected.Packages.Changed = []string{}
			}
			if tc.expected.Packages.None == nil {
				tc.expected.Packages.None = []string{}
			}

			if !reflect.DeepEqual(output, tc.expected) {
				t.Errorf("unexpected output, got %+v, want %+v", output, tc.expected)
			}
		})
	}
}
