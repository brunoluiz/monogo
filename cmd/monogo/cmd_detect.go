package main

import (
	"fmt"
	"path/filepath"

	"github.com/concordalabs/monogo/internal/walker"
	"github.com/concordalabs/monogo/xgit"
	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

type DetectCmd struct {
	Path        string   `help:"Path to detect changes" default:"."`
	Entrypoints []string `help:"Entrypoints to analyze for changes" default:"./cmd/xpdig"`
}

func (r *DetectCmd) Run(c *Context) error {
	changesArr, err := xgit.Diff("main", xgit.WithPath(r.Path))
	if err != nil {
		return fmt.Errorf("failed to load diff: %v", err)
	}
	c.Logger.Info("Changed files", "files", changesArr)
	changed := lo.Map(changesArr, func(change string, _ int) string {
		abs, _ := filepath.Abs(filepath.Join(r.Path, change))
		return abs
	})

	w, err := walker.New(r.Path)
	if err != nil {
		return err
	}

	// NOTE: for each entry point, build a tree based on MAIN branch

	for _, entry := range r.Entrypoints {
		// NOTE: probably need an extra matcher to build the tree on HEAD
		findChanges := changesMatcher{files: changed}
		if err = w.Walk(c.Context, entry, findChanges.Matcher); err != nil {
			return err
		}

		if findChanges.found {
			c.Logger.Info("Changed entrypoint", "entrypoint", entry)
		}

		// NOTE: compare tree size between MAIN x HEAD... deletes/creates would be detected here
	}

	return err
}

type changesMatcher struct {
	files []string
	found bool
}

func (m *changesMatcher) match(b string) func(a string) bool {
	return func(a string) bool {
		return a == b
	}
}

func (m *changesMatcher) Matcher(p *packages.Package) (bool, error) {
	if m.found {
		return true, nil
	}

	_, found := lo.Find(m.files, func(changedFile string) bool {
		if _, ok := lo.Find(p.CompiledGoFiles, m.match(changedFile)); ok {
			return true
		}

		if _, ok := lo.Find(p.EmbedFiles, m.match(changedFile)); ok {
			return true
		}
		return false
	})
	m.found = found
	return m.found, nil
}
