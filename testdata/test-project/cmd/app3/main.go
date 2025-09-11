package main

import (
	"fmt"

	"test-project/pkg/pkgA"
	"test-project/pkg/pkgB"
	"test-project/pkg/shared"
)

func main() {
	fmt.Println("app3")
	fmt.Println(pkgA.A())
	fmt.Println(pkgB.B())
	shared.Log("app3")
}
