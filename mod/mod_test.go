package mod_test

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestModChanges(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("failed: %s", err)
	}

	m, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatalf("failed: %s", err)
	}
	for _, mm := range m.Require {
		fmt.Println(mm.Mod.Path)
		fmt.Println(mm.Mod.Version)
	}
}
