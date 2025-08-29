package hook

import (
	"fmt"

	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

type ChangeDetector struct {
	files     []string
	found     bool
	earlyExit bool
}

func NewChangeDetector(files []string) *ChangeDetector {
	return &ChangeDetector{files: files, earlyExit: false}
}

func (h *ChangeDetector) Found() bool {
	return h.found
}

func (h *ChangeDetector) Do(p *packages.Package) error {
	if h.found && h.earlyExit {
		return fmt.Errorf("%w: early exit on match", ErrEarlyExit)
	}

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
