package walker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

type Walker struct {
	cache interface {
		get(k string) (*packages.Package, bool)
		set(k string, p *packages.Package)
	}
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
		cache:    newCache(),
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
		// NOTE: The pattern (value given by `entry`) must be prefixed with `./` as otherwise it might end up with a package name
		// This becomes a problem when the user configures entrypoints as `cmd/bla` instead of `./cmd/bla`
		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.NeedImports | packages.NeedCompiledGoFiles | packages.NeedDeps | packages.NeedEmbedFiles | packages.NeedEmbedPatterns | packages.NeedName,
			Dir:  w.basePath,
		}, "./"+entry)
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
	if len(pkg.Errors) != 0 {
		return fmt.Errorf("package %s contains errors: %+v", pkg.PkgPath, pkg.Errors)
	}

	if !strings.HasPrefix(pkg.PkgPath, w.module) {
		return nil
	}

	for _, h := range hooks {
		if err := h.Do(pkg); err != nil {
			return err
		}
	}

	// NOTE: Not sure if this cache is correctly placed yet
	if _, found := w.cache.get(pkg.PkgPath); found {
		return nil
	}

	for _, imported := range pkg.Imports {
		if err := w.handlePackage(ctx, imported, hooks...); err != nil {
			return err
		}
	}
	w.cache.set(pkg.PkgPath, pkg)

	return nil
}

// getModuleName extracts the package path of a Go file.
func getModuleName(filePath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(filePath, "go.mod"))
	if err != nil {
		return "", nil
	}

	return modfile.ModulePath(data), nil
}

type cache struct {
	sync.RWMutex
	cache map[string]*packages.Package
}

func newCache() *cache {
	return &cache{RWMutex: sync.RWMutex{}, cache: map[string]*packages.Package{}}
}

func (c *cache) get(k string) (*packages.Package, bool) {
	c.RLock()
	defer c.RUnlock()
	v, ok := c.cache[k]
	return v, ok
}

func (c *cache) set(k string, p *packages.Package) {
	c.Lock()
	defer c.Unlock()
	c.cache[k] = p
}
