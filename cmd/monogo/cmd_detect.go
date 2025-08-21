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
	out, err := r.run(c)
	if err != nil {
		return fmt.Errorf("failed to run detect command: %w", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}

func (r *DetectCmd) run(c *Context) (DetectOutput, error) {
	git, err := xgit.New(xgit.WithPath(r.Path))
	if err != nil {
		return DetectOutput{}, fmt.Errorf("failed to open git repository: %v", err)
	}

	mainBranchTree := map[string][]string{}
	changesArr, err := git.Diff("main")
	if err != nil {
		return DetectOutput{}, fmt.Errorf("failed to load diff: %v", err)
	}

	headHash, headRef, err := git.Head()
	if err != nil {
		return DetectOutput{}, fmt.Errorf("failed to get head ref: %v", err)
	}

	output := DetectOutput{
		Git:         DetectGitOutput{Hash: headHash, Ref: headRef},
		Entrypoints: map[string]EntrypointOutput{},
	}

	if len(changesArr) == 0 {
		return DetectOutput{
			Entrypoints: lo.SliceToMap(r.Entrypoints, func(item string) (string, EntrypointOutput) {
				return item, EntrypointOutput{
					Path:    item,
					Changed: false,
					Reasons: []string{},
				}
			}),
		}, nil
	}

	changed := lo.Map(changesArr, func(change string, _ int) string {
		abs, _ := filepath.Abs(filepath.Join(r.Path, change))
		return abs
	})

	err = git.RunOnRef(r.MainBranch, func() error {
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
	})
	if err != nil {
		return DetectOutput{}, err
	}

	w, err := walker.New(r.Path, c.Logger.WithGroup("walker:ref"))
	if err != nil {
		return DetectOutput{}, err
	}

	for _, entry := range r.Entrypoints {
		changesHook := hook.NewChangeDetector(changed)
		listerHook := hook.NewLister()
		if err = w.Walk(c.Context, entry, changesHook, listerHook); err != nil {
			return DetectOutput{}, err
		}

		reasons := []string{}
		if changesHook.Found() {
			reasons = append(reasons, "files updated")
		}

		if !lo.ElementsMatch(mainBranchTree[entry], listerHook.Files()) {
			reasons = append(reasons, "files created/deleted")
		}

		output.Entrypoints[entry] = EntrypointOutput{
			Path:    entry,
			Changed: len(reasons) > 0,
			Reasons: reasons,
		}
	}

	return output, err
}

type DetectOutput struct {
	Git         DetectGitOutput             `json:"git"`
	Entrypoints map[string]EntrypointOutput `json:"entrypoints"`
}

type DetectGitOutput struct {
	Hash string `json:"hash"`
	Ref  string `json:"ref"`
}

type EntrypointOutput struct {
	Path    string   `json:"path"`
	Changed bool     `json:"changed"`
	Reasons []string `json:"reasons"`
}
