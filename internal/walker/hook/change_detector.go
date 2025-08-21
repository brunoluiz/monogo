package hook

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

var ErrStopCondition = errors.New("stop walker condtion reached")

type ChangeDetector struct {
	srcFiles  []string
	found     bool
	earlyExit bool
}

func NewChangeDetector(files []string) *ChangeDetector {
	return &ChangeDetector{srcFiles: files, earlyExit: false}
}

func (h *ChangeDetector) Found() bool {
	return h.found
}

func (h *ChangeDetector) Do(p *packages.Package) error {
	if h.found && h.earlyExit {
		return fmt.Errorf("%w: early exit on match", ErrStopCondition)
	}

	_, found := lo.Find(h.srcFiles, func(changedFile string) bool {
		if _, ok := lo.Find(p.CompiledGoFiles, h.match(changedFile)); ok {
			return true
		}
		if _, ok := lo.Find(p.EmbedFiles, h.match(changedFile)); ok {
			return true
		}
		return false
	})

	if !h.found {
		h.found = found
	}

	return nil
}

func (h *ChangeDetector) match(b string) func(a string) bool {
	return func(a string) bool {
		return a == b
	}
}
