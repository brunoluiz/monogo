package hook

import (
	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

type ModDetector struct {
	packages []string
	found    bool
}

func NewModDetector(packages []string) *ModDetector {
	return &ModDetector{packages: packages}
}

func (h *ModDetector) Found() bool {
	return h.found
}

func (h *ModDetector) Do(p *packages.Package) error {
	_, h.found = lo.Find(h.packages, func(changedPackage string) bool {
		if _, ok := lo.Find(lo.Keys(p.Imports), match(changedPackage)); ok {
			return true
		}
		return h.found
	})

	return nil
}
