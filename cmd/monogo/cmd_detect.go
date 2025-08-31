package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/concordalabs/monogo/mod"
	"github.com/concordalabs/monogo/walker"
	"github.com/concordalabs/monogo/walker/hook"
	"github.com/concordalabs/monogo/xgit"
	"github.com/samber/lo"
	"golang.org/x/mod/modfile"
)

type DetectCmd struct {
	Path        string   `help:"Path to detect changes" default:"."`
	MainBranch  string   `help:"Main git branch" default:"refs/heads/main"`
	Entrypoints []string `help:"Entrypoints to analyze for changes"`
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

// TODO: somewhere this should check if the mod file is okay / go mod tidy
func (r *DetectCmd) run(c *Context) (DetectOutput, error) {
	git, err := xgit.New(xgit.WithPath(r.Path))
	if err != nil {
		return DetectOutput{}, fmt.Errorf("failed to open git repository: %v", err)
	}

	mainBranchTree := map[string][]string{}
	var mainBranchMod *modfile.File
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
		Stats: DetectStats{
			StartedAt: time.Now(),
			EndedAt:   time.Now(),
		},
	}

	if len(changesArr) == 0 {
		output.Entrypoints = lo.SliceToMap(r.Entrypoints, func(item string) (string, EntrypointOutput) {
			return item, EntrypointOutput{Path: item, Changed: false, Reasons: []string{}}
		})
		return output, nil
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

		_, mainBranchMod, err = mod.Get(mod.WithModDir(r.Path))
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

	_, refMod, err := mod.Get(mod.WithModDir(r.Path))
	if err != nil {
		return DetectOutput{}, err
	}

	modDiff := mod.Diff(mainBranchMod, refMod)

	// In case Golang got updated in go.mod, mark all as changed
	if modDiff.Type == mod.ChangeGolang {
		output.Entrypoints = lo.SliceToMap(r.Entrypoints, func(item string) (string, EntrypointOutput) {
			return item, EntrypointOutput{Path: item, Changed: true, Reasons: []string{"go version changed"}}
		})
		return output, nil
	}

	for _, entry := range r.Entrypoints {
		reasons := []string{}
		changesHook := hook.NewChangeDetector(changed)
		listerHook := hook.NewLister()
		modHook := hook.NewModDetector(modDiff.Packages.All())

		if err = w.Walk(c.Context, entry, changesHook, listerHook, modHook); err != nil {
			return DetectOutput{}, err
		}

		if changesHook.Found() {
			reasons = append(reasons, "files changed")
		}

		if !lo.ElementsMatch(mainBranchTree[entry], listerHook.Files()) {
			reasons = append(reasons, "files created/deleted")
		}

		if modHook.Found() {
			reasons = append(reasons, "dependencies changed")
		}

		output.Entrypoints[entry] = EntrypointOutput{
			Path:    entry,
			Changed: len(reasons) > 0,
			Reasons: reasons,
		}
	}

	output.Stats.EndedAt = time.Now()
	output.Stats.Duration = output.Stats.EndedAt.Sub(output.Stats.StartedAt) / time.Millisecond

	return output, err
}

type DetectOutput struct {
	Git         DetectGitOutput             `json:"git"`
	Stats       DetectStats                 `json:"stats"`
	Entrypoints map[string]EntrypointOutput `json:"entrypoints"`
}

type DetectGitOutput struct {
	Hash string `json:"hash"`
	Ref  string `json:"ref"`
}

type DetectStats struct {
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
	Duration  time.Duration `json:"duration"`
}

type EntrypointOutput struct {
	Path    string   `json:"path"`
	Changed bool     `json:"changed"`
	Reasons []string `json:"reasons"`
}
