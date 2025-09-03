package main

import (
	"fmt"
	"strings"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type VersionCmd struct{}

func (r *VersionCmd) Run(_ *Context) error {
	s := strings.Join([]string{
		"app: monogo",
		fmt.Sprintf("version: %s", version),
		fmt.Sprintf("commit: %s", commit),
		fmt.Sprintf("date: %s", date),
	}, "\n")
	fmt.Println(s)
	return nil
}
