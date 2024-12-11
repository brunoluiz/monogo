package xgit

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// DiffFrom diffs from the current HEAD to the given ref. The output is a list of changes with
// type of operation and patches applied.
// NOTE: in the future option pattern could be applied to allow:
// 1. Change of headCommit to something else (eg: branch)
// 2. Allow other git paths
func Diff(fromRef string, toRef string) (object.Changes, error) {
	r, err := git.PlainOpen(".")
	if err != nil {
		return nil, err
	}

	headRef, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("resolve head ref: %w", err)
	}

	sourceRef, err := r.ResolveRevision(plumbing.Revision(toRef))
	if err != nil {
		return nil, fmt.Errorf("resolve source ref: %w", err)
	}

	headCommit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("head commit ref: %w", err)
	}

	sourceCommit, err := r.CommitObject(*sourceRef)
	if err != nil {
		return nil, fmt.Errorf("source commit ref: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("head tree: %w", err)
	}

	sourceTree, err := sourceCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("source tree: %w", err)
	}

	changes, err := object.DiffTree(sourceTree, headTree)
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}

	return changes, nil
}
