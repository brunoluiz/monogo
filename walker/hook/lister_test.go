package hook_test

import (
	"reflect"
	"testing"

	"github.com/concordalabs/monogo/walker/hook"
	"golang.org/x/tools/go/packages"
)

func TestLister(t *testing.T) {
	pkg1 := &packages.Package{
		ID:              "pkg1",
		CompiledGoFiles: []string{"/path/to/file2.go", "/path/to/file1.go"},
		EmbedFiles:      []string{"/path/to/embed1.txt"},
	}
	pkg2 := &packages.Package{
		ID:              "pkg2",
		CompiledGoFiles: []string{"/path/to/file3.go"},
	}
	pkg1Replacement := &packages.Package{
		ID:              "pkg1",
		CompiledGoFiles: []string{"/path/to/file4.go"},
	}

	testCases := []struct {
		name            string
		pkgsToProcess   []*packages.Package
		expectedFiles   []string
		expectedPkgsLen int
	}{
		{
			name:            "no packages",
			pkgsToProcess:   []*packages.Package{},
			expectedFiles:   []string{},
			expectedPkgsLen: 0,
		},
		{
			name:          "single package",
			pkgsToProcess: []*packages.Package{pkg1},
			expectedFiles: []string{
				"/path/to/embed1.txt",
				"/path/to/file1.go",
				"/path/to/file2.go",
			},
			expectedPkgsLen: 1,
		},
		{
			name:          "multiple packages",
			pkgsToProcess: []*packages.Package{pkg1, pkg2},
			expectedFiles: []string{
				"/path/to/embed1.txt",
				"/path/to/file1.go",
				"/path/to/file2.go",
				"/path/to/file3.go",
			},
			expectedPkgsLen: 2,
		},
		{
			name:          "package replacement",
			pkgsToProcess: []*packages.Package{pkg1, pkg2, pkg1Replacement},
			expectedFiles: []string{
				"/path/to/file3.go",
				"/path/to/file4.go",
			},
			expectedPkgsLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := hook.NewLister()
			if l == nil {
				t.Fatal("NewLister should not return nil")
			}

			for _, pkg := range tc.pkgsToProcess {
				if err := l.Do(pkg); err != nil {
					t.Errorf("unexpected error processing package %s: %v", pkg.ID, err)
				}
			}

			pkgs := l.Packages()
			if len(pkgs) != tc.expectedPkgsLen {
				t.Errorf("expected %d packages, got %d", tc.expectedPkgsLen, len(pkgs))
			}

			files := l.Files()
			if !reflect.DeepEqual(files, tc.expectedFiles) {
				t.Errorf("expected files %v, got %v", tc.expectedFiles, files)
			}
		})
	}
}
