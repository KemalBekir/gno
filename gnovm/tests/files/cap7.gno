package main

func main() {
	var s string
	println("cap", cap(s))
}

// Error:
// main/files/cap7.gno:5:17: unexpected type for cap(): string

// TypeCheckError:
// main/files/cap7.gno:5:21: invalid argument: s (variable of type string) for built-in cap
