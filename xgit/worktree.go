package xgit

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

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
