package xgit

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

type diffConfig struct {
	path string
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
