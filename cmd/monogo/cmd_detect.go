package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/concordalabs/monogo/internal/walker"
	"github.com/concordalabs/monogo/internal/walker/hook"
	"github.com/concordalabs/monogo/xgit"
	"github.com/samber/lo"
)

type DetectCmd struct {
	Path        string   `help:"Path to detect changes" default:"."`
	MainBranch  string   `help:"Main git branch" default:"refs/heads/main"`
	Entrypoints []string `help:"Entrypoints to analyze for changes" default:"./cmd/xpdig"`
}

func (r *DetectCmd) Run(c *Context) error {
	changesArr, err := xgit.Diff("main", xgit.WithPath(r.Path))
	if err != nil {
		return fmt.Errorf("failed to load diff: %v", err)
	}
	c.Logger.DebugContext(c.Context, "Changed files", "files", changesArr)
	changed := lo.Map(changesArr, func(change string, _ int) string {
		abs, _ := filepath.Abs(filepath.Join(r.Path, change))
		return abs
	})

	mainBranchTree := map[string][]string{}

	// TODO: walk through tree to do files in main branch
	// TODO: variable must be customisable
	err = xgit.RunOnRef(r.MainBranch, func() error {
		w, err := walker.New(r.Path, c.Logger.WithGroup("walker:main"))
		if err != nil {
			return err
		}

		for _, entry := range r.Entrypoints {
			listerHook := hook.NewLister()
			if err = w.Walk(c.Context, entry, listerHook); err != nil {
				return err
			}
			mainBranchTree[entry] = listerHook.Files()
		}

		return nil
	}, xgit.WithWorktreePath(r.Path))
	if err != nil {
		return err
	}

	w, err := walker.New(r.Path, c.Logger.WithGroup("walker:ref"))
	if err != nil {
		return err
	}

	output := Res{
		Entrypoints: map[string]EntrypointRes{},
	}
	for _, entry := range r.Entrypoints {
		changesHook := hook.NewChangeDetector(changed)
		listerHook := hook.NewLister()
		if err = w.Walk(c.Context, entry, changesHook, listerHook); err != nil {
			return err
		}

		reasons := []string{}
		if changesHook.Found() {
			reasons = append(reasons, "files updated")
		}

		if !lo.ElementsMatch(mainBranchTree[entry], listerHook.Files()) {
			reasons = append(reasons, "files created/deleted")
		}

		output.Entrypoints[entry] = EntrypointRes{
			Path:    entry,
			Changed: len(reasons) > 0,
			Reasons: reasons,
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return err
}

type Res struct {
	Entrypoints map[string]EntrypointRes `json:"entrypoints"`
}

type EntrypointRes struct {
	Path    string   `json:"path"`
	Changed bool     `json:"changed"`
	Reasons []string `json:"reasons"`
}
