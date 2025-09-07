package monogo_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/brunoluiz/monogo"
	xgit "github.com/brunoluiz/monogo/git"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

func TestDetector_Run(t *testing.T) {
	t.Parallel()

	type fields struct {
		path        string
		mainBranch  string
		entrypoints []string
	}

	tests := []struct {
		name    string
		fields  fields
		prepare func(t *testing.T, repo *git.Repository)
		assert  func(t *testing.T, res monogo.DetectRes)
	}{
		{
			name: "should not detect any changes",
			fields: fields{
				entrypoints: []string{"cmd/app1", "cmd/app2", "cmd/app3"},
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.False(t, res.Entrypoints["cmd/app1"].Changed)
				require.False(t, res.Entrypoints["cmd/app2"].Changed)
				require.False(t, res.Entrypoints["cmd/app3"].Changed)
			},
			prepare: func(t *testing.T, repo *git.Repository) {},
		},
		{
			name: "should detect go version upgrade",
			fields: fields{
				entrypoints: []string{"cmd/app1", "cmd/app2", "cmd/app3"},
			},
			prepare: func(t *testing.T, repo *git.Repository) {
				w, err := repo.Worktree()
				require.NoError(t, err)

				// change go version
				goModPath := filepath.Join(w.Filesystem.Root(), "go.mod")
				data, err := os.ReadFile(goModPath)
				require.NoError(t, err)

				newData := bytes.Replace(data, []byte("go 1.22"), []byte("go 1.23"), 1)
				require.NoError(t, os.WriteFile(goModPath, newData, 0644))

				_, err = w.Add("go.mod")
				require.NoError(t, err)
				_, err = w.Commit("change go version", &git.CommitOptions{})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, res.Entrypoints["cmd/app1"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app1"].Reasons, "go version changed")
				require.True(t, res.Entrypoints["cmd/app2"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app2"].Reasons, "go version changed")
				require.True(t, res.Entrypoints["cmd/app3"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app3"].Reasons, "go version changed")
			},
		},
		{
			name: "should detect external dependency version bump",
			fields: fields{
				entrypoints: []string{"cmd/app1", "cmd/app2", "cmd/app3"},
			},
			prepare: func(t *testing.T, repo *git.Repository) {
				t.Skip()
				w, err := repo.Worktree()
				require.NoError(t, err)

				// bump zap version in go.mod
				goModPath := filepath.Join(w.Filesystem.Root(), "go.mod")
				data, err := os.ReadFile(goModPath)
				require.NoError(t, err)

				newData := bytes.Replace(data, []byte("go.uber.org/zap v1.27.0"), []byte("go.uber.org/zap v1.28.0"), 1)
				require.NoError(t, os.WriteFile(goModPath, newData, 0644))

				_, err = w.Add("go.mod")
				require.NoError(t, err)
				_, err = w.Commit("bump zap version", &git.CommitOptions{})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, res.Entrypoints["cmd/app1"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app1"].Reasons, "dependencies changed")
				require.True(t, res.Entrypoints["cmd/app2"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app2"].Reasons, "dependencies changed")
				require.True(t, res.Entrypoints["cmd/app3"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app3"].Reasons, "dependencies changed")
			},
		},
		{
			name: "should detect new file in internal package",
			fields: fields{
				entrypoints: []string{"cmd/app1", "cmd/app2", "cmd/app3"},
			},
			prepare: func(t *testing.T, repo *git.Repository) {
				w, err := repo.Worktree()
				require.NoError(t, err)

				// add new file
				newFilePath := filepath.Join(w.Filesystem.Root(), "pkg", "pkgA", "new.go")
				require.NoError(t, os.WriteFile(newFilePath, []byte("package pkgA\n\nfunc New() string {\n\treturn \"new\"\n}"), 0644))

				_, err = w.Add(filepath.Join("pkg", "pkgA", "new.go"))
				require.NoError(t, err)
				_, err = w.Commit("add new file", &git.CommitOptions{})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, res.Entrypoints["cmd/app1"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app1"].Reasons, "files created/deleted")
				require.False(t, res.Entrypoints["cmd/app2"].Changed)
				require.True(t, res.Entrypoints["cmd/app3"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app3"].Reasons, "files created/deleted")
			},
		},
		{
			name: "should detect file change in shared package",
			fields: fields{
				entrypoints: []string{"cmd/app1", "cmd/app2", "cmd/app3"},
			},
			prepare: func(t *testing.T, repo *git.Repository) {
				w, err := repo.Worktree()
				require.NoError(t, err)

				// change shared package file
				filePath := filepath.Join(w.Filesystem.Root(), "pkg", "shared", "shared.go")
				content := `package shared

import "go.uber.org/zap"

func Log(msg string) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("CHANGED: " + msg)
}
`
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

				_, err = w.Add(filepath.Join("pkg", "shared", "shared.go"))
				require.NoError(t, err)
				_, err = w.Commit("change shared file", &git.CommitOptions{})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, res.Entrypoints["cmd/app1"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app1"].Reasons, "files changed")
				require.True(t, res.Entrypoints["cmd/app2"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app2"].Reasons, "files changed")
				require.True(t, res.Entrypoints["cmd/app3"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app3"].Reasons, "files changed")
			},
		},
		{
			name: "should detect file change in pkgB",
			fields: fields{
				entrypoints: []string{"cmd/app1", "cmd/app2", "cmd/app3"},
			},
			prepare: func(t *testing.T, repo *git.Repository) {
				w, err := repo.Worktree()
				require.NoError(t, err)

				// change pkgB file
				filePath := filepath.Join(w.Filesystem.Root(), "pkg", "pkgB", "b.go")
				content := `package pkgB

func B() string {
	return "changed"
}
`
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

				_, err = w.Add(filepath.Join("pkg", "pkgB", "b.go"))
				require.NoError(t, err)
				_, err = w.Commit("change pkgB file", &git.CommitOptions{})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.False(t, res.Entrypoints["cmd/app1"].Changed)
				require.True(t, res.Entrypoints["cmd/app2"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app2"].Reasons, "files changed")
				require.True(t, res.Entrypoints["cmd/app3"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app3"].Reasons, "files changed")
			},
		},
		{
			name: "should detect file deletion in internal package",
			fields: fields{
				entrypoints: []string{"cmd/app1", "cmd/app2", "cmd/app3"},
			},
			prepare: func(t *testing.T, repo *git.Repository) {
				w, err := repo.Worktree()
				require.NoError(t, err)

				// delete file
				filePath := filepath.Join(w.Filesystem.Root(), "pkg", "pkgA", "a.go")
				require.NoError(t, os.Remove(filePath))

				_, err = w.Remove(filepath.Join("pkg", "pkgA", "a.go"))
				require.NoError(t, err)
				_, err = w.Commit("delete file", &git.CommitOptions{})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, res.Entrypoints["cmd/app1"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app1"].Reasons, "files created/deleted")
				require.False(t, res.Entrypoints["cmd/app2"].Changed)
				require.True(t, res.Entrypoints["cmd/app3"].Changed)
				require.Contains(t, res.Entrypoints["cmd/app3"].Reasons, "files created/deleted")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel()

			// setup
			tmpDir := t.TempDir()
			repo := setupTestRepo(t, tmpDir)
			w, err := repo.Worktree()
			require.NoError(t, err)

			// initial commit
			_, err = w.Add(".")
			require.NoError(t, err)
			_, err = w.Commit("initial commit", &git.CommitOptions{})
			require.NoError(t, err)

			// ensure main branch reference exists after initial commit
			ref, err := repo.Head()
			require.NoError(t, err)
			err = repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), ref.Hash()))
			require.NoError(t, err)
			err = repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")))
			require.NoError(t, err)
			// checkout to a new branch based on main
			require.NoError(t, w.Checkout(&git.CheckoutOptions{
				Create: true,
				Branch: plumbing.NewBranchReferenceName("test-branch"),
				Hash:   ref.Hash(),
			}))

			// run detector
			g, err := xgit.New(xgit.WithPath(tmpDir))
			require.NoError(t, err)
			d := monogo.NewDetector(tmpDir, tt.fields.entrypoints, string(plumbing.NewBranchReferenceName("main")), slog.Default(), g)

			tt.prepare(t, repo)
			res, err := d.Run(context.Background())
			require.NoError(t, err)
			tt.assert(t, res)
		})
	}
}

func setupTestRepo(t *testing.T, tmpDir string) *git.Repository {
	t.Helper()
	// copy folder
	require.NoError(t, os.CopyFS(tmpDir, os.DirFS("./testdata/test-project")))

	// setup git
	repo, err := git.PlainInitWithOptions(tmpDir, &git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName("main"),
		},
	})
	require.NoError(t, err)
	return repo
}
