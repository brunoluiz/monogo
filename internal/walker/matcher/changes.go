package matcher

import (
	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

type Changes struct {
	files []string
	found bool
}

func NewChanges(files []string) *Changes {
	return &Changes{files: files}
}

func (m *Changes) Found() bool {
	return m.found
}

func (m *Changes) Matcher(p *packages.Package) (bool, error) {
	if m.found {
		return true, nil
	}

	_, found := lo.Find(m.files, func(changedFile string) bool {
		if _, ok := lo.Find(p.CompiledGoFiles, m.match(changedFile)); ok {
			return true
		}

		if _, ok := lo.Find(p.EmbedFiles, m.match(changedFile)); ok {
			return true
		}
		return false
	})
	m.found = found
	return m.found, nil
}

func (m *Changes) match(b string) func(a string) bool {
	return func(a string) bool {
		return a == b
	}
}
