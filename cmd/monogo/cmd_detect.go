package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/brunoluiz/monogo"
	"github.com/brunoluiz/monogo/git"
)

type DetectCmd struct {
	Path        string   `help:"Path to detect changes" default:"."`
	BaseRef     string   `help:"Main git branch" default:"refs/heads/main" help:"Base reference, usually main (e.g., refs/heads/main)"`
	CompareRef  string   `required:"" help:"Compare reference, usually your feature branch (e.g., refs/heads/my-branch)"`
	Entrypoints []string `required:"" help:"Entrypoints to analyze for changes"`
}

func (r *DetectCmd) Run(c *Context) error {
	g, err := git.New(git.WithPath(r.Path))
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	detector := monogo.NewDetector(r.Entrypoints, c.Logger, g,
		monogo.WithBaseRef(r.BaseRef),
		monogo.WithPath(r.Path),
		monogo.WithCompareRef(r.CompareRef),
	)
	out, err := detector.Run(c.Context)
	if err != nil {
		return fmt.Errorf("failed to run detect command: %w", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}
