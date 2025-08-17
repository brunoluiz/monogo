package main

import (
	"github.com/concordalabs/monogo/xgit"
)

type DetectCmd struct {
	Path string `help:"Path to detect changes" default:"."`
}

func (r *DetectCmd) Run(c *Context) error {
	x, err := xgit.Diff("main", xgit.WithPath(r.Path))
	c.Logger.Info("Detected changes", "changes", x)
	return err
}
