package hook

import (
	"slices"

	"golang.org/x/tools/go/packages"
)

type Lister struct {
	embedFiles []string
	goFiles    []string
	packages   []string
}

func NewLister() *Lister {
	return &Lister{
		embedFiles: []string{},
		goFiles:    []string{},
		packages:   []string{},
	}
}

func (h *Lister) Files() []string {
	files := []string{}
	files = append(files, h.embedFiles...)
	files = append(files, h.goFiles...)
	slices.Sort(files)
	return files
}

func (h *Lister) Do(p *packages.Package) error {
	h.embedFiles = append(h.embedFiles, p.EmbedFiles...)
	h.goFiles = append(h.goFiles, p.CompiledGoFiles...)
	h.packages = append(h.packages, p.ID)
	return nil
}
