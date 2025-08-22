package xgit

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

type Git struct {
	repo *git.Repository
}

type gitConfig struct {
	path string
}

type WithOpt func(*gitConfig)

func WithPath(path string) func(*gitConfig) {
	return func(c *gitConfig) {
		c.path = path
	}
}

func New(opts ...WithOpt) (*Git, error) {
	cfg := gitConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	r, err := git.PlainOpen(cfg.path)
	if err != nil {
		return nil, err
	}

	return &Git{
		repo: r,
	}, nil
}

// Head returns current HEAD details
func (g *Git) Head() (string, string, error) {
	headRef, err := g.repo.Head()
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve head ref: %w", err)
	}

	return headRef.Hash().String(), string(headRef.Name()), nil
}

func (g *Git) RunOnRef(ref string, cb func() error) error {
	currentBranch, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to resolve head ref: %w", err)
	}

	wt, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to resolve worktree: %w", err)
	}

	defer func() {
		if err := wt.Checkout(&git.CheckoutOptions{Branch: currentBranch.Name()}); err != nil {
			fmt.Print(err)
		}
	}()

	if err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.ReferenceName(ref)}); err != nil {
		return fmt.Errorf("failed to checkout: %w", err)
	}

	return cb()
}

// Diff diffs from the current HEAD to the given ref. The output is a list of changes with
// type of operation and patches applied.
func (g *Git) Diff(compareRef string) ([]string, error) {
	headRef, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve head ref: %w", err)
	}

	sourceRef, err := g.repo.ResolveRevision(plumbing.Revision(compareRef))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source ref: %w", err)
	}

	headCommit, err := g.repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get head commit ref: %w", err)
	}

	sourceCommit, err := g.repo.CommitObject(*sourceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get source commit ref: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get head tree: %w", err)
	}

	sourceTree, err := sourceCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("falted to get source tree: %w", err)
	}

	changes, err := object.DiffTree(sourceTree, headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff: %w", err)
	}

	var files []string // nolint: prealloc
	for _, change := range changes {
		// Ignore deleted files
		action, err := change.Action()
		if err != nil {
			return nil, fmt.Errorf("failed to get diff action: %w", err)
		}

		if action == merkletrie.Delete {
			continue
		}

		name := change.To.Name
		if change.From.Name != "" {
			name = change.From.Name
		}
		files = append(files, name)
	}

	return files, nil
}
