package hook

import (
	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

type ChangeDetector struct {
	files []string
	found bool
}

func NewChangeDetector(files []string) *ChangeDetector {
	return &ChangeDetector{files: files}
}

func (h *ChangeDetector) Found() bool {
	return h.found
}

func (h *ChangeDetector) Do(p *packages.Package) error {
	_, h.found = lo.Find(h.files, func(changedFile string) bool {
		if _, ok := lo.Find(p.CompiledGoFiles, match(changedFile)); ok {
			return true
		}
		if _, ok := lo.Find(p.EmbedFiles, match(changedFile)); ok {
			return true
		}
		return h.found
	})

	return nil
}
