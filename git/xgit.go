package git

import (
	"fmt"
	"sort"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

type Git struct {
	repo *git.Repository
}

type DiffResult struct {
	Created []string
	Updated []string
	Deleted []string
}

func (d DiffResult) All() []string {
	all := make([]string, 0, len(d.Created)+len(d.Updated)+len(d.Deleted))
	all = append(all, d.Created...)
	all = append(all, d.Updated...)
	all = append(all, d.Deleted...)
	sort.Strings(all)
	return all
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

	currentRef, err := g.repo.Head()
	if err != nil {
		return err
	}
	defer func() {
		wt.Checkout(&git.CheckoutOptions{Hash: currentRef.Hash()})
	}()

	rev, err := g.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return fmt.Errorf("failed to resolve revision %s: %w", ref, err)
	}

	if err = wt.Checkout(&git.CheckoutOptions{Hash: *rev}); err != nil {
		return fmt.Errorf("failed to checkout: %w", err)
	}

	return cb()
}

// Diff diffs from the given ref to the compare ref. The output is a DiffResult with
// Created, Updated, and Deleted files.
func (g *Git) Diff(fromRef, compareRef string) (DiffResult, error) {
	fromRefResolved, err := g.repo.ResolveRevision(plumbing.Revision(fromRef))
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to resolve from ref %s: %w", fromRef, err)
	}

	compareRefResolved, err := g.repo.ResolveRevision(plumbing.Revision(compareRef))
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to resolve compare ref %s: %w", compareRef, err)
	}

	fromCommit, err := g.repo.CommitObject(*fromRefResolved)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to get from commit ref: %w", err)
	}

	compareCommit, err := g.repo.CommitObject(*compareRefResolved)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to get compare commit ref: %w", err)
	}

	fromTree, err := fromCommit.Tree()
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to get from tree: %w", err)
	}

	compareTree, err := compareCommit.Tree()
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to get compare tree: %w", err)
	}

	changes, err := object.DiffTree(compareTree, fromTree)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to diff: %w", err)
	}

	result := DiffResult{Created: []string{}, Updated: []string{}, Deleted: []string{}}
	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to get diff action: %w", err)
		}

		name := change.To.Name
		if change.From.Name != "" {
			name = change.From.Name
		}

		switch action {
		case merkletrie.Insert:
			result.Created = append(result.Created, name)
		case merkletrie.Delete:
			result.Deleted = append(result.Deleted, change.From.Name)
		case merkletrie.Modify:
			result.Updated = append(result.Updated, name)
		}
	}

	return result, nil
}
