package main

import "gno.land/p/moul/debug"

func main() {
	var d debug.Debug
	d.Log("hello world!")
	d.Log("foobar")
	println("---")
	println(d.Render(""))
	println("---")
	println(d.Render("?debug=1"))
	println("---")
}

// Output:
// ---
//
// ---
// <details><summary>debug</summary>
//
// ### Logs
// - hello world!
// - foobar
// ### Metadata
// | Key | Value |
// | --- | --- |
// | `std.CurrentRealm().PkgPath()` |  |
// | `std.CurrentRealm().Address()` | g1wymu47drhr0kuq2098m792lytgtj2nyx77yrsm |
// | `std.PreviousRealm().PkgPath()` |  |
// | `std.PreviousRealm().Address()` | g1wymu47drhr0kuq2098m792lytgtj2nyx77yrsm |
// | `std.ChainHeight()` | 123 |
// | `time.Now().Format(time.RFC3339)` | 2009-02-13T23:31:30Z |
//
// </details>
//
// ---
