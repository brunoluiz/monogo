package mod

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/samber/lo"
	"golang.org/x/mod/modfile"
)

type WithOpt func(c *options)

type options struct {
	path string
}

func WithModDir(path string) WithOpt {
	return func(c *options) {
		c.path = filepath.Join(path, "go.mod")
	}
}

// TODO: make the go.mod path customisable
func Get(opts ...WithOpt) (string, *modfile.File, error) {
	c := options{path: "go.mod"}
	for _, opt := range opts {
		opt(&c)
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return "", nil, fmt.Errorf("error to open go module file: %w", err)
	}

	m, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", nil, fmt.Errorf("error to parse go module file: %w", err)
	}

	return modfile.ModulePath(data), m, nil
}

type ChangeType int

const (
	ChangeUnknown ChangeType = iota
	ChangeNone
	ChangePackages
	ChangeGolang
	ChangeGolangToolchain
)

type ChangedPackages struct {
	Added   []string
	Deleted []string
	Changed []string
	None    []string
}

func (c ChangedPackages) All() []string {
	return lo.Uniq(append(append(c.Added, c.Deleted...), c.Changed...))
}

type Output struct {
	Type     ChangeType
	Packages ChangedPackages
}

func Diff(leftMod, rightMod *modfile.File) Output {
	if lo.FromPtr(leftMod.Go).Version != lo.FromPtr(rightMod.Go).Version {
		return Output{Type: ChangeGolang}
	}

	if lo.FromPtr(leftMod.Toolchain).Name != lo.FromPtr(rightMod.Toolchain).Name {
		return Output{Type: ChangeGolangToolchain}
	}

	left, right, changed, none := []string{}, []string{}, []string{}, []string{}
	leftPkgs := lo.SliceToMap(leftMod.Require, func(item *modfile.Require) (string, *modfile.Require) {
		return item.Mod.Path, item
	})
	rightPkgs := lo.SliceToMap(rightMod.Require, func(item *modfile.Require) (string, *modfile.Require) {
		return item.Mod.Path, item
	})

	for leftKey, leftPkg := range leftPkgs {
		rightPkg, found := rightPkgs[leftKey]
		if !found {
			left = append(left, leftPkg.Mod.Path)
			continue
		}

		// NOTE: I think it only needs to be done once
		if leftPkg.Mod.Version != rightPkg.Mod.Version {
			changed = append(changed, leftPkg.Mod.Path)
			continue
		}

		none = append(none, leftPkg.Mod.Path)
	}

	for rightKey, rightPkg := range rightPkgs {
		_, found := leftPkgs[rightKey]
		if !found {
			right = append(right, rightPkg.Mod.Path)
			continue
		}
	}

	if len(left) > 0 || len(right) > 0 || len(changed) > 0 {
		return Output{Type: ChangePackages, Packages: ChangedPackages{
			Added:   right,
			Deleted: left,
			None:    lo.Uniq(none),
			Changed: lo.Uniq(changed),
		}}
	}

	return Output{Type: ChangeNone}
}
