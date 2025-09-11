package monogo

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/brunoluiz/monogo/git"
	"github.com/brunoluiz/monogo/mod"
	"github.com/brunoluiz/monogo/walker"
	"github.com/brunoluiz/monogo/walker/hook"
	"github.com/samber/lo"
	"golang.org/x/mod/modfile"
)

type ChangeReason string

const (
	ChangedFilesReason         ChangeReason = "files changed"
	CreatedDeletedFilesReasons ChangeReason = "files created/deleted"
	DependenciesChangedReason  ChangeReason = "dependencies changed"
	GoVersionChangedReason     ChangeReason = "go version changed"
	NoChangesReason            ChangeReason = "no changes"
)

type DetectRes struct {
	Skipped     bool                           `json:"skipped"`
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
	Path    string         `json:"path"`
	Changed bool           `json:"changed"`
	Reasons []ChangeReason `json:"reasons"`
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
	headHash, headRef, err := r.Git.Head()
	if err != nil {
		return DetectRes{}, fmt.Errorf("failed to get head ref: %w", err)
	}

	output := DetectRes{
		Git:         DetectGitRes{Hash: headHash, Ref: headRef},
		Stats:       DetectStatsRes{StartedAt: time.Now(), EndedAt: time.Now()},
		Entrypoints: map[string]DetectEntrypointRes{},
	}

	changes, err := r.Git.Diff(r.MainBranch)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failed to load diff: %w", err)
	}

	if len(changes) == 0 {
		output.Entrypoints = lo.SliceToMap(r.Entrypoints, func(item string) (string, DetectEntrypointRes) {
			return item, DetectEntrypointRes{Path: item, Changed: false, Reasons: []ChangeReason{NoChangesReason}}
		})
		return output, nil
	}

	mainInfo, err := r.getMainBranchInfo(ctx)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failure while getting main tree info: %w", err)
	}

	diffInfo, err := r.getDiffInfo(ctx, mainInfo, changes)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failure while getting diff info: %w", err)
	}

	output.Entrypoints = diffInfo.entrypoints
	output.Stats.EndedAt = time.Now()
	output.Stats.Duration = output.Stats.EndedAt.Sub(output.Stats.StartedAt) / time.Millisecond

	return output, err
}

type mainBranchInfo struct {
	filesByEntrypoint map[string][]string
	modfile           *modfile.File
}

func (r *Detector) getMainBranchInfo(ctx context.Context) (mainBranchInfo, error) {
	info := mainBranchInfo{filesByEntrypoint: map[string][]string{}}

	err := r.Git.RunOnRef(r.MainBranch, func() error {
		w, err := walker.New(r.Path, r.Logger.WithGroup("walker:main"))
		if err != nil {
			return err
		}

		_, info.modfile, err = mod.Get(mod.WithModDir(r.Path))
		if err != nil {
			return err
		}

		for _, entry := range r.Entrypoints {
			listerHook := hook.NewLister()
			if err = w.Walk(ctx, entry, listerHook); err != nil {
				return err
			}
			info.filesByEntrypoint[entry] = listerHook.Files()
		}

		return nil
	})
	if err != nil {
		return mainBranchInfo{}, err
	}
	return info, nil
}

type diffInfo struct {
	entrypoints map[string]DetectEntrypointRes
}

func (r *Detector) getDiffInfo(ctx context.Context, mainInfo mainBranchInfo, changes []string) (diffInfo, error) {
	w, err := walker.New(r.Path, r.Logger.WithGroup("walker:ref"))
	if err != nil {
		return diffInfo{}, err
	}

	changesByAbsPath := lo.Map(changes, func(change string, _ int) string {
		// nolint
		abs, _ := filepath.Abs(filepath.Join(r.Path, change))
		return abs
	})

	_, refMod, err := mod.Get(mod.WithModDir(r.Path))
	if err != nil {
		return diffInfo{}, err
	}

	modDiff := mod.Diff(mainInfo.modfile, refMod)

	// In case Golang got updated in go.mod, mark all as changed
	if modDiff.Type == mod.ChangeGolang {
		return diffInfo{
			entrypoints: lo.SliceToMap(r.Entrypoints, func(item string) (string, DetectEntrypointRes) {
				return item, DetectEntrypointRes{Path: item, Changed: true, Reasons: []ChangeReason{GoVersionChangedReason}}
			}),
		}, nil
	}

	info := diffInfo{entrypoints: map[string]DetectEntrypointRes{}}
	for _, entry := range r.Entrypoints {
		reasons := []ChangeReason{}
		changesHook := hook.NewChangeDetector(changesByAbsPath)
		listerHook := hook.NewLister()
		modHook := hook.NewModDetector(modDiff.Packages.All())

		if err = w.Walk(ctx, entry, changesHook, listerHook, modHook); err != nil {
			return diffInfo{}, err
		}

		if changesHook.Found() {
			reasons = append(reasons, ChangedFilesReason)
		}

		if !lo.ElementsMatch(mainInfo.filesByEntrypoint[entry], listerHook.Files()) {
			reasons = append(reasons, CreatedDeletedFilesReasons)
		}

		if modHook.Found() {
			reasons = append(reasons, DependenciesChangedReason)
		}

		info.entrypoints[entry] = DetectEntrypointRes{
			Path:    entry,
			Changed: len(reasons) > 0,
			Reasons: reasons,
		}
	}

	return info, nil
}
