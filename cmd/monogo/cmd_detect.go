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

	for _, entry := range r.Entrypoints {
		w, err := walker.New(r.Path)
		if err != nil {
			return err
		}

		matcher := changedPackageMatcher{changedPkgs: changedPkgs}
		if err = w.Walk(c.Context, entry, matcher.Match); err != nil {
			return err
		}

		if matcher.found {
			c.Logger.Info("Changed entrypoint", "entrypoint", entry)
		}
	}

	return err
}

type changedPackageMatcher struct {
	changedPkgs []*packages.Package
	found       bool
}

func (m *changedPackageMatcher) Match(p *packages.Package) (bool, error) {
	if m.found {
		return true, nil
	}

	_, ok := lo.Find(m.changedPkgs, func(changedPkg *packages.Package) bool {
		return changedPkg.ID == p.ID
	})
	m.found = ok
	return ok, nil
}
