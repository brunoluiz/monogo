package xgit

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
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

func (g *Git) Head() (string, string, error) {
	headRef, err := g.repo.Head()
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve head ref: %w", err)
	}

	return headRef.Hash().String(), string(headRef.Name()), nil
}
