package main

func main() {
	a := 2
	s := map[string]float32{"k": 1 << a}
	println(s["k"])
}

// Error:
// main/shift_d41.gno:5:7-38: operator << not defined on: Float32Kind

// TypeCheckError:
// main/shift_d41.gno:5:31: invalid operation: shifted operand 1 (type float32) must be integer
