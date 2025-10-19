package git

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

// Ref returns details for a specific ref
func (g *Git) Ref(ref string) (string, string, error) {
	refResolved, err := g.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve ref %s: %w", ref, err)
	}

	return refResolved.String(), ref, nil
}

func (g *Git) RunOnRef(ref string, cb func() error) error {
	wt, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to resolve worktree: %w", err)
	}

	if err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.ReferenceName(ref)}); err != nil {
		return fmt.Errorf("failed to checkout: %w", err)
	}

	return cb()
}

// Diff diffs from the given ref to the compare ref. The output is a list of changes with
// type of operation and patches applied.
func (g *Git) Diff(fromRef, compareRef string) ([]string, error) {
	fromRefResolved, err := g.repo.ResolveRevision(plumbing.Revision(fromRef))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve from ref %s: %w", fromRef, err)
	}

	compareRefResolved, err := g.repo.ResolveRevision(plumbing.Revision(compareRef))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve compare ref %s: %w", compareRef, err)
	}

	fromCommit, err := g.repo.CommitObject(*fromRefResolved)
	if err != nil {
		return nil, fmt.Errorf("failed to get from commit ref: %w", err)
	}

	compareCommit, err := g.repo.CommitObject(*compareRefResolved)
	if err != nil {
		return nil, fmt.Errorf("failed to get compare commit ref: %w", err)
	}

	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get from tree: %w", err)
	}

	compareTree, err := compareCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get compare tree: %w", err)
	}

	changes, err := object.DiffTree(compareTree, fromTree)
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
			files = append(files, change.From.Name)
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
