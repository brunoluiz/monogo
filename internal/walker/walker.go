package walker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

type Walker struct {
	cache    map[string]*packages.Package
	basePath string
	module   string
}

func New(basePath string) (*Walker, error) {
	module, err := getModuleName(basePath)
	if err != nil {
		return nil, fmt.Errorf("base path might not be a module: %w", err)
	}

	return &Walker{
		cache:    make(map[string]*packages.Package),
		basePath: basePath,
		module:   module,
	}, nil
}

type Hook interface {
	Do(p *packages.Package) (err error)
}

func (w *Walker) Walk(ctx context.Context, entry string, hooks ...Hook) error {
	return w.walk(ctx, entry, hooks...)
}

func (w *Walker) walk(ctx context.Context, entry string, hooks ...Hook) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		// Load all packages in the codebase
		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.NeedImports | packages.NeedCompiledGoFiles | packages.NeedDeps | packages.NeedEmbedFiles | packages.NeedEmbedPatterns | packages.NeedName | packages.NeedExportFile,
			Dir:  w.basePath,
		}, entry)
		if err != nil {
			return fmt.Errorf("failed to load packages: %v", err)
		}

		// Iterate through all packages to find dependencies
		for _, pkg := range pkgs {
			for _, imported := range pkg.Imports {
				if _, found := w.cache[imported.PkgPath]; found {
					continue
				}

				if !strings.HasPrefix(imported.PkgPath, w.module) {
					continue
				}

				// fmt.Printf("%s\n", imported)
				w.cache[imported.PkgPath] = imported

				for _, h := range hooks {
					if err := h.Do(imported); err != nil {
						return err
					}
				}

				if err := w.walk(ctx, imported.PkgPath, hooks...); err != nil {
					return fmt.Errorf("error finding dependent files for package %s: %s", pkg.PkgPath, err)
				}
			}
		}

		return nil
	}
}

// getModuleName extracts the package path of a Go file.
func getModuleName(filePath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(filePath, "go.mod"))
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		return "", nil
	}

	return modfile.ModulePath(data), nil
}
