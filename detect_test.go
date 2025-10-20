package monogo_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brunoluiz/monogo"
	xgit "github.com/brunoluiz/monogo/git"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

func findEntrypoint(entrypoints []monogo.DetectEntrypointRes, path string) *monogo.DetectEntrypointRes {
	for _, entry := range entrypoints {
		if entry.Path == path {
			return &entry
		}
	}
	return nil
}

// nolint: funlen
func TestDetector_Run(t *testing.T) {
	t.Parallel()

	// Test author for all commits
	testAuthor := &object.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time.Now(),
	}

	type fields struct {
		entrypoints   []string
		showUnchanged bool
	}

	tests := []struct {
		name    string
		fields  fields
		prepare func(t *testing.T, w *git.Worktree)
		assert  func(t *testing.T, res monogo.DetectRes)
	}{
		{
			name: "should not detect any changes",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: false,
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.False(t, findEntrypoint(res.Entrypoints, "cmd/app1").Changed)
				require.False(t, findEntrypoint(res.Entrypoints, "cmd/app2").Changed)
				require.False(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Empty(t, res.Git.Files.Created.All)
				require.Empty(t, res.Git.Files.Updated.All)
				require.Empty(t, res.Git.Files.Deleted.All)
			},
			prepare: func(_ *testing.T, _ *git.Worktree) {},
		},
		{
			name: "should detect go version upgrade",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: false,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				// change go version
				targetFile := "go.mod"
				worktreeTargetPath := filepath.Join(w.Filesystem.Root(), targetFile)
				data, err := os.ReadFile(worktreeTargetPath)
				require.NoError(t, err)

				newData := bytes.Replace(data, []byte("go 1.22"), []byte("go 1.23"), 1)
				require.NoError(t, os.WriteFile(worktreeTargetPath, newData, 0o600))

				_, err = w.Add(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("change go version", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app1").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app1").Reasons, monogo.GoVersionChangedReason)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app2").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app2").Reasons, monogo.GoVersionChangedReason)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app3").Reasons, monogo.GoVersionChangedReason)
				require.Contains(t, res.Git.Files.Updated.All, "go.mod")
				require.NotContains(t, res.Git.Files.Updated.Go, "go.mod")
			},
		},
		{
			name: "should detect external dependency version bump",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: false,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				t.Skip()

				// bump zap version in go.mod
				targetFile := "go.mod"
				targetWorktreePath := filepath.Join(w.Filesystem.Root(), targetFile)
				data, err := os.ReadFile(targetWorktreePath)
				require.NoError(t, err)

				newData := bytes.Replace(data, []byte("go.uber.org/zap v1.27.0"), []byte("go.uber.org/zap v1.28.0"), 1)
				require.NoError(t, os.WriteFile(targetWorktreePath, newData, 0o600))

				_, err = w.Add(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("bump zap version", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app1").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app1").Reasons, monogo.DependenciesChangedReason)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app2").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app2").Reasons, monogo.DependenciesChangedReason)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app3").Reasons, monogo.DependenciesChangedReason)
			},
		},
		{
			name: "should detect new file in internal package",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: false,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				// add new file
				targetFile := filepath.Join("pkg", "pkgA", "new.go")
				targetWorktreePath := filepath.Join(w.Filesystem.Root(), targetFile)
				require.NoError(t, os.WriteFile(targetWorktreePath, []byte("package pkgA\n\nfunc New() string {\n\treturn \"new\"\n}"), 0o600))

				var err error
				_, err = w.Add(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("add new file", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			}, assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app1").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app1").Reasons, monogo.CreatedDeletedFilesReasons)
				require.Nil(t, findEntrypoint(res.Entrypoints, "cmd/app2"))
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app3").Reasons, monogo.CreatedDeletedFilesReasons)
			},
		},
		{
			name: "should detect file change in shared package",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: false,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				// change shared package file
				targetFile := filepath.Join("pkg", "shared", "shared.go")
				targetWorktreePath := filepath.Join(w.Filesystem.Root(), targetFile)
				content := `package shared

import "go.uber.org/zap"

func Log(msg string) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("CHANGED: " + msg)
}
`
				require.NoError(t, os.WriteFile(targetWorktreePath, []byte(content), 0o600))

				_, err := w.Add(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("change shared file", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app1").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app1").Reasons, monogo.ChangedFilesReason)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app2").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app2").Reasons, monogo.ChangedFilesReason)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app3").Reasons, monogo.ChangedFilesReason)
				require.Contains(t, res.Git.Files.Updated.All, "pkg/shared/shared.go")
				require.Contains(t, res.Git.Files.Updated.Go, "pkg/shared/shared.go")
			},
		},
		{
			name: "should detect file change in pkgB",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: false,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				// change pkgB file
				targetFile := filepath.Join("pkg", "pkgB", "b.go")
				targetWorktreeFile := filepath.Join(w.Filesystem.Root(), targetFile)
				content := `package pkgB

func B() string {
	return "changed"
}
`
				require.NoError(t, os.WriteFile(targetWorktreeFile, []byte(content), 0o600))

				_, err := w.Add(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("change pkgB file", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.Nil(t, findEntrypoint(res.Entrypoints, "cmd/app1"))
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app2").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app2").Reasons, monogo.ChangedFilesReason)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app3").Reasons, monogo.ChangedFilesReason)
				require.Contains(t, res.Git.Files.Updated.All, "pkg/pkgB/b.go")
				require.Contains(t, res.Git.Files.Updated.Go, "pkg/pkgB/b.go")
			},
		},
		{
			name: "should detect file deletion in internal package",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: false,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				// delete file
				targetFile := filepath.Join("pkg", "pkgA", "deleteme.go")
				targetWorktreeFile := filepath.Join(w.Filesystem.Root(), targetFile)
				require.NoError(t, os.Remove(targetWorktreeFile))

				_, err := w.Remove(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("delete file", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app1").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app1").Reasons, monogo.CreatedDeletedFilesReasons)
				require.Nil(t, findEntrypoint(res.Entrypoints, "cmd/app2"))
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app3").Reasons, monogo.CreatedDeletedFilesReasons)
				require.Contains(t, res.Git.Files.Deleted.All, "pkg/pkgA/deleteme.go")
				require.Contains(t, res.Git.Files.Deleted.Go, "pkg/pkgA/deleteme.go")
			},
		},
		{
			name: "should detect new cmd that does not exist in main branch",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3", "cmd/app4"},
				showUnchanged: false,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				// create new cmd directory and main.go that doesn't exist in main branch
				targetDir := filepath.Join("cmd", "app4")
				targetDirWorktree := filepath.Join(w.Filesystem.Root(), targetDir)
				require.NoError(t, os.MkdirAll(targetDirWorktree, 0o755))

				targetFile := filepath.Join(targetDir, "main.go")
				targetFileWorktree := filepath.Join(w.Filesystem.Root(), targetFile)
				content := `package main

import (
	"fmt"
	"test-project/pkg/shared"
)

func main() {
	shared.Log("Hello from app4")
	fmt.Println("App4 running")
}
`
				require.NoError(t, os.WriteFile(targetFileWorktree, []byte(content), 0o600))

				_, err := w.Add(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("add new cmd app4", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				// existing apps should not be included since unchanged
				require.Nil(t, findEntrypoint(res.Entrypoints, "cmd/app1"))
				require.Nil(t, findEntrypoint(res.Entrypoints, "cmd/app2"))
				require.Nil(t, findEntrypoint(res.Entrypoints, "cmd/app3"))
				// new app should be detected as changed (since it doesn't exist in main)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app4").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app4").Reasons, monogo.CreatedDeletedFilesReasons)
				require.Contains(t, res.Git.Files.Created.All, "cmd/app4/main.go")
				require.Contains(t, res.Git.Files.Created.Go, "cmd/app4/main.go")
			},
		},
		{
			name: "should show unchanged entrypoints when showUnchanged is true",
			fields: fields{
				entrypoints:   []string{"cmd/app1", "cmd/app2", "cmd/app3"},
				showUnchanged: true,
			},
			prepare: func(t *testing.T, w *git.Worktree) {
				// add new file that affects app1 and app3 but not app2
				targetFile := filepath.Join("pkg", "pkgA", "new.go")
				targetWorktreePath := filepath.Join(w.Filesystem.Root(), targetFile)
				require.NoError(t, os.WriteFile(targetWorktreePath, []byte("package pkgA\n\nfunc New() string {\n\treturn \"new\"\n}"), 0o600))

				var err error
				_, err = w.Add(targetFile)
				require.NoError(t, err)
				_, err = w.Commit("add new file", &git.CommitOptions{
					Author: testAuthor,
				})
				require.NoError(t, err)
			},
			assert: func(t *testing.T, res monogo.DetectRes) {
				// app1 and app3 should be changed
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app1").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app1").Reasons, monogo.CreatedDeletedFilesReasons)
				require.True(t, findEntrypoint(res.Entrypoints, "cmd/app3").Changed)
				require.Contains(t, findEntrypoint(res.Entrypoints, "cmd/app3").Reasons, monogo.CreatedDeletedFilesReasons)
				// app2 should be unchanged but included
				require.False(t, findEntrypoint(res.Entrypoints, "cmd/app2").Changed)
				require.Empty(t, findEntrypoint(res.Entrypoints, "cmd/app2").Reasons)
				require.Contains(t, res.Git.Files.Created.All, "pkg/pkgA/new.go")
				require.Contains(t, res.Git.Files.Created.Go, "pkg/pkgA/new.go")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel()

			// setup folder
			tmpDir := filepath.Join("./tmp", t.Name())
			require.NoError(t, os.RemoveAll(tmpDir))
			require.NoError(t, os.MkdirAll(tmpDir, 0o755))
			require.NoError(t, os.CopyFS(tmpDir, os.DirFS("./testdata/test-project")))

			// setup git
			repo, err := git.PlainInitWithOptions(tmpDir, &git.PlainInitOptions{
				InitOptions: git.InitOptions{DefaultBranch: plumbing.NewBranchReferenceName("main")},
			})
			require.NoError(t, err)
			w, err := repo.Worktree()
			require.NoError(t, err)

			// initial commit
			_, err = w.Add(".")
			require.NoError(t, err)
			_, err = w.Commit("initial commit", &git.CommitOptions{
				Author: testAuthor,
			})
			require.NoError(t, err)

			// checkout to a new branch based on main
			ref, err := repo.Head()
			require.NoError(t, err)
			b := plumbing.NewBranchReferenceName("test-branch")
			require.NoError(t, w.Checkout(&git.CheckoutOptions{
				Create: true,
				Branch: b,
				Hash:   ref.Hash(),
			}))

			// run detector
			g, err := xgit.New(xgit.WithPath(tmpDir))
			require.NoError(t, err)
			d := monogo.NewDetector(tt.fields.entrypoints, slog.Default(), g,
				monogo.WithPath(tmpDir),
				monogo.WithBaseRef(string(plumbing.NewBranchReferenceName("main"))),
				monogo.WithCompareRef(string(b)),
				monogo.WithShowUnchanged(tt.fields.showUnchanged),
			)

			tt.prepare(t, w)
			res, err := d.Run(context.Background())
			require.NoError(t, err)
			tt.assert(t, res)
		})
	}
}
