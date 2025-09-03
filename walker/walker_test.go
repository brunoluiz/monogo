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
		name          string
		entry         string
		expectedPaths []string
	}{
		{
			name:          "entry from pkgC",
			entry:         "test/project/pkgC",
			expectedPaths: []string{"test/project/pkgC", "test/project/pkgA", "test/project/pkgB"},
		},
		{
			name:          "entry from pkgA",
			entry:         "test/project/pkgA",
			expectedPaths: []string{"test/project/pkgA", "test/project/pkgB"},
		},
		{
			name:          "entry from pkgB",
			entry:         "test/project/pkgB",
			expectedPaths: []string{"test/project/pkgB"},
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

			var gotPaths []string
			for _, p := range hook.calledWith {
				gotPaths = append(gotPaths, p.PkgPath)
			}

			sort.Strings(gotPaths)
			sort.Strings(tc.expectedPaths)

			if !reflect.DeepEqual(gotPaths, tc.expectedPaths) {
				t.Errorf("unexpected packages, got %+v, want %+v", gotPaths, tc.expectedPaths)
			}
		})
	}
}
