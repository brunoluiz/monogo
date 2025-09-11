package main

import (
	"fmt"

	"test-project/pkg/pkgA"
	"test-project/pkg/shared"
)

func main() {
	fmt.Println("app1")
	fmt.Println(pkgA.A())
	shared.Log("app1")
}
