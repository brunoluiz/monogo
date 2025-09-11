package walker_test

import (
	"context"
	"log/slog"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/brunoluiz/monogo/walker"
	"golang.org/x/tools/go/packages"
)

type mockHook struct {
	calledWith []*packages.Package
}

func (m *mockHook) Do(p *packages.Package) error {
	m.calledWith = append(m.calledWith, p)
	return nil
}

func TestWalker_Walk(t *testing.T) {
	testCases := []struct {
		name             string
		entry            string
		expectedPkgPaths []string
	}{
		{
			name:             "entry from pkgC",
			entry:            "pkgC",
			expectedPkgPaths: []string{"test/project/pkgC", "test/project/pkgA", "test/project/pkgB"},
		},
		{
			name:             "entry from pkgA",
			entry:            "pkgA",
			expectedPkgPaths: []string{"test/project/pkgA", "test/project/pkgB"},
		},
		{
			name:             "entry from pkgB",
			entry:            "./pkgB",
			expectedPkgPaths: []string{"test/project/pkgB"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			w, err := walker.New("./testdata/project", logger)
			if err != nil {
				t.Fatalf("failed to create walker: %s", err)
			}

			hook := &mockHook{}
			err = w.Walk(context.Background(), tc.entry, hook)
			if err != nil {
				t.Fatalf("failed to walk: %s", err)
			}

			var gotPkgPaths []string
			for _, p := range hook.calledWith {
				gotPkgPaths = append(gotPkgPaths, p.PkgPath)
			}

			sort.Strings(gotPkgPaths)
			sort.Strings(tc.expectedPkgPaths)

			if !reflect.DeepEqual(gotPkgPaths, tc.expectedPkgPaths) {
				t.Errorf("unexpected packages, got %+v, want %+v", gotPkgPaths, tc.expectedPkgPaths)
			}
		})
	}
}
