package hook_test

import (
	"testing"

	"github.com/brunoluiz/monogo/walker/hook"
	"golang.org/x/tools/go/packages"
)

func TestChangeDetector(t *testing.T) {
	files := []string{"/path/to/file1.go", "/path/to/file2.go", "/path/to/asset.txt"}

	testCases := []struct {
		name          string
		pkg           *packages.Package
		expectedFound bool
	}{
		{
			name:          "no match",
			pkg:           &packages.Package{ID: "pkg1", CompiledGoFiles: []string{"/path/to/file3.go"}},
			expectedFound: false,
		},
		{
			name:          "match on go file",
			pkg:           &packages.Package{ID: "pkg2", CompiledGoFiles: []string{"/path/to/file1.go"}},
			expectedFound: true,
		},
		{
			name:          "match on embedded file",
			pkg:           &packages.Package{ID: "pkg4", EmbedFiles: []string{"/path/to/asset.txt"}},
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cd := hook.NewChangeDetector(files)
			err := cd.Do(tc.pkg)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if cd.Found() != tc.expectedFound {
				t.Errorf("expected found to be %v, but got %v", tc.expectedFound, cd.Found())
			}
		})
	}
}
