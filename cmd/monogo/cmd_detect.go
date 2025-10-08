package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/brunoluiz/monogo"
	"github.com/brunoluiz/monogo/git"
	"github.com/samber/lo"
)

type DetectCmd struct {
	Path        string   `help:"Path to detect changes" default:"."`
	MainBranch  string   `help:"Main git branch" default:"refs/heads/main"`
	Entrypoints []string `help:"Entrypoints to analyze for changes"`
	OmitNoChanges bool     `help:"Omit entrypoints that have no changes"`
}

func (r *DetectCmd) Run(c *Context) error {
	g, err := git.New(git.WithPath(r.Path))
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	detector := monogo.NewDetector(r.Entrypoints, c.Logger, g,
		monogo.WithMainBranch(r.MainBranch),
		monogo.WithPath(r.Path),
	)
	out, err := detector.Run(c.Context)
	if err != nil {
		return fmt.Errorf("failed to run detect command: %w", err)
	}

	if r.OmitNoChanges {
		out.Entrypoints = lo.Filter(out.Entrypoints, func(item monogo.DetectEntrypointRes, index int) bool {
			return item.Changed
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}
