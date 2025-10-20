package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/brunoluiz/monogo"
	"github.com/brunoluiz/monogo/git"
)

type DetectCmd struct {
	Path          string   `help:"Path to detect changes" default:"."`
	BaseRef       string   `default:"refs/heads/main" help:"Base reference, usually main (e.g., refs/heads/main)"`
	CompareRef    string   `required:"" help:"Compare reference, usually your feature branch (e.g., refs/heads/my-branch)"`
	Entrypoints   []string `required:"" help:"Entrypoints to analyze for changes"`
	ShowUnchanged bool     `help:"Show unchanged entrypoints in the output" default:"false"`
	Output        string   `help:"Output format: json or github" default:"json" enum:"json,github"`
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
		monogo.WithShowUnchanged(r.ShowUnchanged),
	)
	out, err := detector.Run(c.Context)
	if err != nil {
		return fmt.Errorf("failed to run detect command: %w", err)
	}

	switch r.Output {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
			return fmt.Errorf("failed to encode output: %w", err)
		}
	case "github":
		if err := outputGitHub(out); err != nil {
			return fmt.Errorf("failed to output github format: %w", err)
		}
	default:
		return fmt.Errorf("unknown output format: %s", r.Output)
	}

	return nil
}

func outputGitHub(out monogo.DetectRes) error {
	jsonBytes, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	entrypointsBytes, err := json.Marshal(out.Entrypoints)
	if err != nil {
		return fmt.Errorf("failed to marshal entrypoints: %w", err)
	}

	fmt.Printf("json=%s\n", string(jsonBytes))
	fmt.Printf("entrypoints=%s\n", string(entrypointsBytes))
	fmt.Printf("impacted_go_files=%s\n", strings.Join(out.Git.Files.Impacted.Go, ","))
	fmt.Printf("changed=%t\n", out.Changed)

	return nil
}
