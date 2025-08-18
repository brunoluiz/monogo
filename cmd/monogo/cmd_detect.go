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
	changedPkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
		Dir:  r.Path,
	}, lo.Map(changesArr, func(change string, _ int) string {
		return filepath.Join(r.Path, filepath.Dir(change))
	})...)
	if err != nil {
		return fmt.Errorf("failed to load packages: %v", err)
	}

	w, err := walker.New(r.Path)
	if err != nil {
		return err
	}

	// NOTE: for each entry point, build a tree based on MAIN branch

	for _, entry := range r.Entrypoints {
		// NOTE: probably need an extra matcher to build the tree on HEAD
		changedPkg := changedPackageMatcher{changedPkgs: changedPkgs}
		if err = w.Walk(c.Context, entry, changedPkg.Matcher); err != nil {
			return err
		}

		if changedPkg.changed {
			c.Logger.Info("Changed entrypoint", "entrypoint", entry)
		}

		// NOTE: compare tree size between MAIN x HEAD... deletes/creates would be detected here
	}

	return err
}

type changedPackageMatcher struct {
	changedPkgs []*packages.Package
	changed     bool
}

func (m *changedPackageMatcher) Matcher(p *packages.Package) (bool, error) {
	if m.changed {
		return true, nil
	}

	_, ok := lo.Find(m.changedPkgs, func(changedPkg *packages.Package) bool {
		return changedPkg.ID == p.ID
	})
	m.changed = ok
	return ok, nil
}
