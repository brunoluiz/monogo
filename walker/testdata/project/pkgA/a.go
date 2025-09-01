package pkgA

import "test/project/pkgB"

func PkgA() {
	pkgB.PkgB()
}
