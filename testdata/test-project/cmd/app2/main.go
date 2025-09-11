package main

import (
	"fmt"

	"test-project/pkg/pkgB"
	"test-project/pkg/shared"
)

func main() {
	fmt.Println("app2")
	fmt.Println(pkgB.B())
	shared.Log("app2")
}
