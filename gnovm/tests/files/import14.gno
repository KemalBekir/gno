// PKGPATH: github.com/gnolang/gno/_test/foo

package foo

import (
	"github.com/gnolang/gno/_test/foo/internal"
	"github.com/gnolang/gno/_test/foo/internal/aa"
)

func main() {
	println(internal.A, internal.B)
	println(aa.A)
}

// Output:
// 123 456
// 456
