package matcher

import (
	"golang.org/x/tools/go/packages"
)

type List struct {
	files []string
}

func NewList() *List {
	return &List{files: []string{}}
}

func (m *List) List() []string {
	return m.files
}

func (m *List) Matcher(p *packages.Package) (bool, error) {
	// FIXME: consider embeded files
	m.files = append(m.files, p.EmbedFiles...)
	m.files = append(m.files, p.CompiledGoFiles...)
	return false, nil
}
