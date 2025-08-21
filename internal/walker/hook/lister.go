package hook

import (
	"slices"

	"golang.org/x/tools/go/packages"
)

type Lister struct {
	packages map[string]*packages.Package
}

func NewLister() *Lister {
	return &Lister{
		packages: map[string]*packages.Package{},
	}
}

func (h *Lister) Files() []string {
	files := []string{}
	for _, pkg := range h.packages {
		files = append(files, pkg.CompiledGoFiles...)
		files = append(files, pkg.EmbedFiles...)
	}

	slices.Sort(files)
	return files
}

func (h *Lister) Packages() map[string]*packages.Package {
	return h.packages
}

func (h *Lister) Do(p *packages.Package) error {
	h.packages[p.ID] = p
	return nil
}
