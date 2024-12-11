package xgit_test

import (
	"testing"

	"github.com/concordalabs/monogo/xgit"
	"github.com/davecgh/go-spew/spew"
)

func TestDiff(t *testing.T) {
	spew.Dump(xgit.Diff("master", "head"))
}
