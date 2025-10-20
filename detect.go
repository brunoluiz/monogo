package monogo

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
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
	Changed     bool                  `json:"changed"`
	Git         DetectGitRes          `json:"git"`
	Stats       DetectStatsRes        `json:"stats"`
	Entrypoints []DetectEntrypointRes `json:"entrypoints"`
}

type DetectGitRes struct {
	Hash  string              `json:"hash"`
	Ref   string              `json:"ref"`
	Files DetectGitChangesRes `json:"files"`
}

type DetectGitChangesRes struct {
	Created DetectFileTypeRes `json:"created"`
	Updated DetectFileTypeRes `json:"updated"`
	Deleted DetectFileTypeRes `json:"deleted"`
}

type DetectFileTypeRes struct {
	All []string `json:"all"`
	Go  []string `json:"go"`
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
	Path          string
	BaseRef       string
	CompareRef    string
	Entrypoints   []string
	Logger        *slog.Logger
	Git           *git.Git
	ShowUnchanged bool
}

type WithDetectOpt func(*detectorConfig)

type detectorConfig struct {
	path          string
	baseRef       string
	compareRef    string
	showUnchanged bool
}

func WithPath(path string) func(*detectorConfig) {
	return func(d *detectorConfig) {
		d.path = path
	}
}

func WithBaseRef(branch string) func(*detectorConfig) {
	return func(d *detectorConfig) {
		d.baseRef = branch
	}
}

func WithCompareRef(branch string) func(*detectorConfig) {
	return func(d *detectorConfig) {
		d.compareRef = branch
	}
}

func WithShowUnchanged(show bool) func(*detectorConfig) {
	return func(d *detectorConfig) {
		d.showUnchanged = show
	}
}

func NewDetector(
	entrypoints []string,
	logger *slog.Logger,
	g *git.Git,
	opts ...WithDetectOpt,
) *Detector {
	cfg := detectorConfig{
		baseRef: "refs/heads/main",
		path:    ".",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Detector{
		Path:          cfg.path,
		BaseRef:       cfg.baseRef,
		CompareRef:    cfg.compareRef,
		Entrypoints:   entrypoints,
		Logger:        logger,
		Git:           g,
		ShowUnchanged: cfg.showUnchanged,
	}
}

func (r *Detector) Run(ctx context.Context) (DetectRes, error) {
	refHash, refName, err := r.Git.Ref(r.CompareRef)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failed to get ref: %w", err)
	}

	res := DetectRes{
		Git: DetectGitRes{
			Hash: refHash,
			Ref:  refName,
			Files: DetectGitChangesRes{
				Created: DetectFileTypeRes{Go: []string{}, All: []string{}},
				Deleted: DetectFileTypeRes{Go: []string{}, All: []string{}},
				Updated: DetectFileTypeRes{Go: []string{}, All: []string{}},
			},
		},
		Stats:       DetectStatsRes{StartedAt: time.Now(), EndedAt: time.Now()},
		Entrypoints: []DetectEntrypointRes{},
	}

	diffResult, err := r.Git.Diff(r.CompareRef, r.BaseRef)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failed to load diff: %w", err)
	}

	r.populateFilesFromChanges(&res.Git.Files, diffResult)

	if len(diffResult.All()) == 0 {
		res.Entrypoints = lo.Map(r.Entrypoints, func(item string, _ int) DetectEntrypointRes {
			return DetectEntrypointRes{Path: item, Changed: false, Reasons: []ChangeReason{NoGitChangesReason}}
		})
		return res, nil
	}

	mainInfo, err := r.getMainBranchInfo(ctx)
	if err != nil {
		return DetectRes{}, fmt.Errorf("failure while getting main tree info: %w", err)
	}

	diffInfo, err := r.getDiffInfo(ctx, mainInfo, diffResult.All())
	if err != nil {
		return DetectRes{}, fmt.Errorf("failure while getting diff info: %w", err)
	}

	res.Entrypoints = diffInfo.entrypoints
	res.Stats.EndedAt = time.Now()
	res.Stats.Duration = res.Stats.EndedAt.Sub(res.Stats.StartedAt) / time.Millisecond
	res.Changed = lo.SomeBy(res.Entrypoints, func(item DetectEntrypointRes) bool {
		return item.Changed
	})
	return res, err
}

func (r *Detector) populateFilesFromChanges(files *DetectGitChangesRes, changes git.DiffResult) {
	isGolangFile := func(file string, _ int) bool { return strings.HasSuffix(file, ".go") }

	files.Created.All = changes.Created
	files.Updated.All = changes.Updated
	files.Deleted.All = changes.Deleted
	files.Created.Go = lo.Filter(changes.Created, isGolangFile)
	files.Updated.Go = lo.Filter(changes.Updated, isGolangFile)
	files.Deleted.Go = lo.Filter(changes.Deleted, isGolangFile)
}

type mainBranchInfo struct {
	filesByEntrypoint map[string][]string
	modfile           *modfile.File
}

func (r *Detector) getMainBranchInfo(ctx context.Context) (mainBranchInfo, error) {
	info := mainBranchInfo{filesByEntrypoint: map[string][]string{}}
	err := r.Git.RunOnRef(r.BaseRef, func() error {
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
	entrypoints []DetectEntrypointRes
}

func (r *Detector) getDiffInfo(ctx context.Context, mainInfo mainBranchInfo, changes []string) (diffInfo, error) {
	info := diffInfo{entrypoints: []DetectEntrypointRes{}}
	err := r.Git.RunOnRef(r.CompareRef, func() error {
		w, err := walker.New(r.Path, r.Logger.WithGroup("walker:ref"))
		if err != nil {
			return err
		}

		changesByAbsPath := lo.Map(changes, func(change string, _ int) string {
			// nolint
			abs, _ := filepath.Abs(filepath.Join(r.Path, change))
			return abs
		})

		_, refMod, err := mod.Get(mod.WithModDir(r.Path))
		if err != nil {
			return err
		}

		// In case Golang got updated, mark all as changed
		modDiff := mod.Diff(mainInfo.modfile, refMod)
		if modDiff.Type == mod.ChangeGolang {
			info = diffInfo{
				entrypoints: lo.Map(r.Entrypoints, func(item string, _ int) DetectEntrypointRes {
					return DetectEntrypointRes{Path: item, Changed: true, Reasons: []ChangeReason{GoVersionChangedReason}}
				}),
			}
			return nil
		}

		// Runs each entrypoint walker with go routines: you must test it with `-race` enabled
		eg, ctx := errgroup.WithContext(ctx)
		rw := sync.RWMutex{}
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
				changed := len(reasons) > 0
				if changed || r.ShowUnchanged {
					info.entrypoints = append(info.entrypoints, DetectEntrypointRes{
						Path:    entry,
						Changed: changed,
						Reasons: reasons,
					})
				}
				return nil
			})
		}

		return eg.Wait()
	})

	return info, err
}
