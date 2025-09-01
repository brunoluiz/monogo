package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/concordalabs/monogo"
	"github.com/concordalabs/monogo/git"
)

type DetectCmd struct {
	Path        string   `help:"Path to detect changes" default:"."`
	MainBranch  string   `help:"Main git branch" default:"refs/heads/main"`
	Entrypoints []string `help:"Entrypoints to analyze for changes"`
}

func (r *DetectCmd) Run(c *Context) error {
	g, err := git.New(git.WithPath(r.Path))
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	detector := monogo.NewDetector(r.Path, r.Entrypoints, r.MainBranch, c.Logger, g)
	out, err := detector.Run(c.Context)
	if err != nil {
		return fmt.Errorf("failed to run detect command: %w", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}
