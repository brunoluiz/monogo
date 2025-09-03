package hook_test

import (
	"testing"

	"github.com/brunoluiz/monogo/walker/hook"
	"golang.org/x/tools/go/packages"
)

func TestModDetector(t *testing.T) {
	pkgs := []string{"example.com/a", "example.com/b"}

	testCases := []struct {
		name          string
		pkg           *packages.Package
		initialFound  bool
		expectedFound bool
	}{
		{
			name:          "no match",
			pkg:           &packages.Package{ID: "pkg1", Imports: map[string]*packages.Package{"example.com/c": {}}},
			expectedFound: false,
		},
		{
			name:          "match",
			pkg:           &packages.Package{ID: "pkg2", Imports: map[string]*packages.Package{"example.com/a": {}}},
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			md := hook.NewModDetector(pkgs)
			err := md.Do(tc.pkg)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if md.Found() != tc.expectedFound {
				t.Errorf("expected found to be %v, but got %v", tc.expectedFound, md.Found())
			}
		})
	}
}
