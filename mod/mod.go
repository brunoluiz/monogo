package mod

import (
	"fmt"
	"os"

	"github.com/samber/lo"
	"golang.org/x/mod/modfile"
)

// TODO: make the go.mod path customisable
func Get() (*modfile.File, error) {
	data, err := os.ReadFile("go.mod")
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

type ChangedPackages struct {
	Added   []string
	Deleted []string
	Changed []string
	None    []string
}

type Output struct {
	Type     ChangeType
	Packages ChangedPackages
}

func Diff(leftMod, rightMod *modfile.File) Output {
	diffVersion := false
	if leftMod.Go != nil && rightMod.Go != nil {
		diffVersion = leftMod.Go.Version != rightMod.Go.Version
	} else if leftMod.Go != rightMod.Go {
		diffVersion = true
	}

	diffToolchain := false
	if leftMod.Toolchain != nil && rightMod.Toolchain != nil {
		diffToolchain = leftMod.Toolchain.Name != rightMod.Toolchain.Name
	} else if leftMod.Toolchain != rightMod.Toolchain {
		diffToolchain = true
	}

	if diffVersion || diffToolchain {
		return Output{Type: ChangeTooling}
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
