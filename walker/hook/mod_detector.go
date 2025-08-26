package hook

import (
	"fmt"

	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

type ModDetector struct {
	packages  []string
	found     bool
	earlyExit bool
}

func NewModDetector(packages []string) *ModDetector {
	return &ModDetector{packages: packages, earlyExit: false}
}

func (h *ModDetector) Found() bool {
	return h.found
}

func (h *ModDetector) Do(p *packages.Package) error {
	if h.found && h.earlyExit {
		return fmt.Errorf("%w: early exit on match", ErrStopCondition)
	}

	_, h.found = lo.Find(h.packages, func(changedPackage string) bool {
		if _, ok := lo.Find(lo.Keys(p.Imports), match(changedPackage)); ok {
			return true
		}
		return h.found
	})

	return nil
}
