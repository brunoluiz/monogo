package monogo

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/brunoluiz/monogo/git"
	"github.com/brunoluiz/monogo/mod"
	"github.com/brunoluiz/monogo/walker"
	"github.com/brunoluiz/monogo/walker/hook"
	"github.com/samber/lo"
	"golang.org/x/mod/modfile"
	"golang.org/x/sync/errgroup"
)

type ChangeReason string

const (
	ChangedFilesReason         ChangeReason = "files changed"
	CreatedDeletedFilesReasons ChangeReason = "files created/deleted"
	DependenciesChangedReason  ChangeReason = "dependencies changed"
	GoVersionChangedReason     ChangeReason = "go version changed"
	NoGitChangesReason         ChangeReason = "no git changes"
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

type WithDetectOpt func(*detectorConfig)

type detectorConfig struct {
	path       string
	mainBranch string
}

func WithPath(path string) func(*detectorConfig) {
	return func(d *detectorConfig) {
		d.path = path
	}
}

func WithMainBranch(branch string) func(*detectorConfig) {
	return func(d *detectorConfig) {
		d.mainBranch = branch
	}
}

func NewDetector(
	entrypoints []string,
	logger *slog.Logger,
	g *git.Git,
	opts ...WithDetectOpt,
) *Detector {
	cfg := detectorConfig{
		mainBranch: "main",
		path:       ".",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Detector{
		Path:        cfg.path,
		MainBranch:  cfg.mainBranch,
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

	res := DetectRes{
		Git:         DetectGitRes{Hash: headHash, Ref: headRef},
		Stats:       DetectStatsRes{StartedAt: time.Now(), EndedAt: time.Now()},
		Entrypoints: map[string]DetectEntrypointRes{},
	}

	changes, err := r.Git.Diff(r.MainBranch)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failed to load diff: %w", err)
	}

	if len(changes) == 0 {
		res.Entrypoints = lo.SliceToMap(r.Entrypoints, func(item string) (string, DetectEntrypointRes) {
			return item, DetectEntrypointRes{Path: item, Changed: false, Reasons: []ChangeReason{NoGitChangesReason}}
		})
		return res, nil
	}

	mainInfo, err := r.getMainBranchInfo(ctx)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failure while getting main tree info: %w", err)
	}

	diffInfo, err := r.getDiffInfo(ctx, mainInfo, changes)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failure while getting diff info: %w", err)
	}

	res.Entrypoints = diffInfo.entrypoints
	res.Stats.EndedAt = time.Now()
	res.Stats.Duration = res.Stats.EndedAt.Sub(res.Stats.StartedAt) / time.Millisecond
	return res, err
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

		// Runs each entrypoint walker with go routines: you must test it with `-race` enabled
		eg, ctx := errgroup.WithContext(ctx)
		rw := sync.RWMutex{}
		for _, entry := range r.Entrypoints {
			entry := entry
			eg.Go(func() error {
				// Walks through all packages for this entry
				listerHook := hook.NewLister()
				err := w.Walk(ctx, entry, listerHook)

				// Write operations to shared memory below
				rw.Lock()
				defer rw.Unlock()

				if err != nil {
					// If the entrypoint doesn't exist in main branch, treat as empty
					r.Logger.Debug("entrypoint not found in main branch", "entry", entry, "error", err)
					info.filesByEntrypoint[entry] = []string{}
				} else {
					info.filesByEntrypoint[entry] = listerHook.Files()
				}
				return nil
			})
		}

		return eg.Wait()
	})

	return info, err
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

	// In case Golang got updated, mark all as changed
	modDiff := mod.Diff(mainInfo.modfile, refMod)
	if modDiff.Type == mod.ChangeGolang {
		return diffInfo{
			entrypoints: lo.SliceToMap(r.Entrypoints, func(item string) (string, DetectEntrypointRes) {
				return item, DetectEntrypointRes{Path: item, Changed: true, Reasons: []ChangeReason{GoVersionChangedReason}}
			}),
		}, nil
	}

	// Runs each entrypoint walker with go routines: you must test it with `-race` enabled
	eg, ctx := errgroup.WithContext(ctx)
	rw := sync.RWMutex{}
	info := diffInfo{entrypoints: map[string]DetectEntrypointRes{}}
	for _, entry := range r.Entrypoints {
		entry := entry
		eg.Go(func() error {
			// Walks through all packages for this entry
			reasons := []ChangeReason{}
			changesHook := hook.NewChangeDetector(changesByAbsPath)
			listerHook := hook.NewLister()
			modHook := hook.NewModDetector(modDiff.Packages.All())
			if err := w.Walk(ctx, entry, changesHook, listerHook, modHook); err != nil {
				return err
			}

			// Assertions and reason mapping
			if changesHook.Found() {
				reasons = append(reasons, ChangedFilesReason)
			}
			if !lo.ElementsMatch(mainInfo.filesByEntrypoint[entry], listerHook.Files()) {
				reasons = append(reasons, CreatedDeletedFilesReasons)
			}
			if modHook.Found() {
				reasons = append(reasons, DependenciesChangedReason)
			}

			// Write operations to shared memory below
			rw.Lock()
			defer rw.Unlock()
			info.entrypoints[entry] = DetectEntrypointRes{
				Path:    entry,
				Changed: len(reasons) > 0,
				Reasons: reasons,
			}
			return nil
		})
	}

	return info, eg.Wait()
}
