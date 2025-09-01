package monogo

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/concordalabs/monogo/git"
	"github.com/concordalabs/monogo/mod"
	"github.com/concordalabs/monogo/walker"
	"github.com/concordalabs/monogo/walker/hook"
	"github.com/samber/lo"
	"golang.org/x/mod/modfile"
)

type DetectRes struct {
	Git         DetectGitRes                   `json:"git"`
	Stats       DetectStatsRes                 `json:"stats"`
	Entrypoints map[string]DetectEntrypointRes `json:"entrypoints"`
}

type DetectGitRes struct {
	Hash string `json:"hash"`
	Ref  string `json:"ref"`
}

type DetectStatsRes struct {
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
	Duration  time.Duration `json:"duration"`
}

type DetectEntrypointRes struct {
	Path    string   `json:"path"`
	Changed bool     `json:"changed"`
	Reasons []string `json:"reasons"`
}

type Detector struct {
	Path        string
	MainBranch  string
	Entrypoints []string
	Logger      *slog.Logger
	Git         *git.Git
}

func NewDetector(
	path string,
	entrypoints []string,
	mainBranch string,
	logger *slog.Logger,
	g *git.Git,
) *Detector {
	if mainBranch == "" {
		mainBranch = "main"
	}

	return &Detector{
		Path:        path,
		MainBranch:  mainBranch,
		Entrypoints: entrypoints,
		Logger:      logger,
		Git:         g,
	}
}

func (r *Detector) Run(ctx context.Context) (DetectRes, error) {
	mainBranchTree := map[string][]string{}
	var mainBranchMod *modfile.File
	changesArr, err := r.Git.Diff("main")
	if err != nil {
		return DetectRes{}, fmt.Errorf("failed to load diff: %v", err)
	}

	headHash, headRef, err := r.Git.Head()
	if err != nil {
		return DetectRes{}, fmt.Errorf("failed to get head ref: %v", err)
	}

	output := DetectRes{
		Git:         DetectGitRes{Hash: headHash, Ref: headRef},
		Stats:       DetectStatsRes{StartedAt: time.Now(), EndedAt: time.Now()},
		Entrypoints: map[string]DetectEntrypointRes{},
	}

	if len(changesArr) == 0 {
		output.Entrypoints = lo.SliceToMap(r.Entrypoints, func(item string) (string, DetectEntrypointRes) {
			return item, DetectEntrypointRes{Path: item, Changed: false, Reasons: []string{}}
		})
		return output, nil
	}

	changed := lo.Map(changesArr, func(change string, _ int) string {
		abs, _ := filepath.Abs(filepath.Join(r.Path, change))
		return abs
	})

	if err = r.Git.RunOnRef(r.MainBranch, func() error {
		w, err := walker.New(r.Path, r.Logger.WithGroup("walker:main"))
		if err != nil {
			return err
		}

		_, mainBranchMod, err = mod.Get(mod.WithModDir(r.Path))
		if err != nil {
			return err
		}

		for _, entry := range r.Entrypoints {
			listerHook := hook.NewLister()
			if err = w.Walk(ctx, entry, listerHook); err != nil {
				return err
			}
			mainBranchTree[entry] = listerHook.Files()
		}

		return nil
	}); err != nil {
		return DetectRes{}, err
	}

	w, err := walker.New(r.Path, r.Logger.WithGroup("walker:ref"))
	if err != nil {
		return DetectRes{}, err
	}

	_, refMod, err := mod.Get(mod.WithModDir(r.Path))
	if err != nil {
		return DetectRes{}, err
	}

	modDiff := mod.Diff(mainBranchMod, refMod)

	// In case Golang got updated in go.mod, mark all as changed
	if modDiff.Type == mod.ChangeGolang {
		output.Entrypoints = lo.SliceToMap(r.Entrypoints, func(item string) (string, DetectEntrypointRes) {
			return item, DetectEntrypointRes{Path: item, Changed: true, Reasons: []string{"go version changed"}}
		})
		return output, nil
	}

	for _, entry := range r.Entrypoints {
		reasons := []string{}
		changesHook := hook.NewChangeDetector(changed)
		listerHook := hook.NewLister()
		modHook := hook.NewModDetector(modDiff.Packages.All())

		if err = w.Walk(ctx, entry, changesHook, listerHook, modHook); err != nil {
			return DetectRes{}, err
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

		output.Entrypoints[entry] = DetectEntrypointRes{
			Path:    entry,
			Changed: len(reasons) > 0,
			Reasons: reasons,
		}
	}

	output.Stats.EndedAt = time.Now()
	output.Stats.Duration = output.Stats.EndedAt.Sub(output.Stats.StartedAt) / time.Millisecond

	return output, err
}
