package walker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

type Walker struct {
	cache    map[string]*packages.Package
	logger   *slog.Logger
	basePath string
	module   string
}

func New(basePath string, logger *slog.Logger) (*Walker, error) {
	module, err := getModuleName(basePath)
	if err != nil {
		return nil, fmt.Errorf("base path might not be a module: %w", err)
	}

	return &Walker{
		cache:    make(map[string]*packages.Package),
		logger:   logger,
		basePath: basePath,
		module:   module,
	}, nil
}

type Hook interface {
	Do(p *packages.Package) (err error)
}

func (w *Walker) Walk(ctx context.Context, entry string, hooks ...Hook) error {
	w.logger.Debug("Starting walk", slog.String("entry", entry))
	return w.walk(ctx, entry, hooks...)
}

func (w *Walker) walk(ctx context.Context, entry string, hooks ...Hook) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		// Load all packages in the codebase
		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.NeedImports | packages.NeedCompiledGoFiles | packages.NeedDeps | packages.NeedEmbedFiles | packages.NeedEmbedPatterns | packages.NeedName,
			Dir:  w.basePath,
		}, entry)
		if err != nil {
			return fmt.Errorf("failed to load packages: %w", err)
		}

		// Iterate through all packages to find dependencies
		for _, pkg := range pkgs {
			if err := w.handlePackage(ctx, pkg, hooks...); err != nil {
				return err
			}
		}

		return nil
	}
}

func (w *Walker) handlePackage(
	ctx context.Context,
	pkg *packages.Package,
	hooks ...Hook,
) error {
	if _, found := w.cache[pkg.PkgPath]; found {
		return nil
	}

	if !strings.HasPrefix(pkg.PkgPath, w.module) {
		return nil
	}
	w.cache[pkg.PkgPath] = pkg

	for _, h := range hooks {
		if err := h.Do(pkg); err != nil {
			return err
		}
	}

	for _, imported := range pkg.Imports {
		if err := w.handlePackage(ctx, imported, hooks...); err != nil {
			return err
		}
	}

	return nil
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
