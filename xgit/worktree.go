package xgit

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type worktreeConfig struct {
	path string
}

type WithWorktreeExecOpt func(*worktreeConfig)

func WithWorktreePath(path string) func(*worktreeConfig) {
	return func(c *worktreeConfig) {
		c.path = path
	}
}

func RunOnRef(ref string, cb func() error, opts ...WithWorktreeExecOpt) error {
	cfg := worktreeConfig{
		path: ".",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	r, err := git.PlainOpen(cfg.path)
	if err != nil {
		return err
	}

	currentBranch, err := r.Head()
	if err != nil {
		return fmt.Errorf("failed to resolve head ref: %w", err)
	}

	wt, err := r.Worktree()
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
