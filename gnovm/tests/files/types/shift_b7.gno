package main

import "fmt"

func main() {
	x := 2
	r := float32(1<<x + 2)
	println(r)
	fmt.Printf("%T\n", r)
}

// Error:
// main/shift_b7.gno:7:7-24: operator << not defined on: Float32Kind

// TypeCheckError:
// main/shift_b7.gno:7:15: invalid operation: shifted operand 1 (type float32) must be integer
