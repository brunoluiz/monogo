package mod

import (
	"fmt"
	"os"

	"github.com/samber/lo"
	"golang.org/x/mod/modfile"
)

func ChangedPackages(path string) (*modfile.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error to open go module file: %w", err)
	}

	m, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("error to parse go module file: %w", err)
	}

	return m, nil
}

type ChangeType int

const (
	ChangeUnknown ChangeType = iota
	ChangeNone
	ChangePackages
	ChangeTooling
)

type ChangeOutput struct {
	Type     ChangeType
	Packages []string
}

func Diff(a, b *modfile.File) ChangeOutput {
	diffVersion := a.Go.Version != b.Go.Version
	diffToolchain := a.Toolchain.Name != b.Toolchain.Name
	if diffVersion || diffToolchain {
		return ChangeOutput{Type: ChangeTooling}
	}

	x, y := lo.Difference(a.Require, b.Require)
	// TODO: implement the package list
	if len(x) > 0 || len(y) > 0 {
		/*
		 * get mod details from main
		 * get mod details from branch
		 * if Go changed: rebuild all
		 * if Required module was updated: find on walker
		 * if Required module was created/deleted: we dont care, it will show up as a file change, but probably will show up on walker
		 */
		return ChangeOutput{Type: ChangePackages, Packages: []string{}}
	}

	return ChangeOutput{Type: ChangeNone}
}
