package main

import (
	"fmt"
	"path/filepath"

	"github.com/concordalabs/monogo/internal/walker"
	"github.com/concordalabs/monogo/internal/walker/matcher"
	"github.com/concordalabs/monogo/xgit"
	"github.com/samber/lo"
)

type DetectCmd struct {
	Path        string   `help:"Path to detect changes" default:"."`
	MainBranch  string   `help:"Main git branch" default:"main"`
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

	mainBranchTree := map[string][]string{}

	// TODO: walk through tree to do files in main branch
	// TODO: variable must be customisable
	err = xgit.WorktreeExec(r.MainBranch, func() error {
		w, err := walker.New(r.Path)
		if err != nil {
			return err
		}

		for _, entry := range r.Entrypoints {
			// NOTE: probably need an extra matcher to build the tree on HEAD
			listPkgs := matcher.NewList()
			if err = w.Walk(c.Context, entry, listPkgs.Matcher); err != nil {
				return err
			}
			mainBranchTree[entry] = listPkgs.List()
		}

		return nil
	}, xgit.WithWorktreePath(r.Path))
	if err != nil {
		return err
	}

	// NOTE: for each entry point, build a tree based on MAIN branch
	w, err := walker.New(r.Path)
	if err != nil {
		return err
	}

	for _, entry := range r.Entrypoints {
		// NOTE: probably need an extra matcher to build the tree on HEAD
		findChanges := matcher.NewChanges(changed)
		listPkgs := matcher.NewList()
		if err = w.Walk(c.Context, entry, findChanges.Matcher, listPkgs.Matcher); err != nil {
			return err
		}

		if findChanges.Found() {
			c.Logger.Info("Changed entrypoint due to updated files", "entrypoint", entry)
		}

		if !lo.ElementsMatch(mainBranchTree[entry], listPkgs.List()) {
			c.Logger.Info("Changed entrypoint due to created/deleted files", "entrypoint", entry)
		}
	}

	return err
}
