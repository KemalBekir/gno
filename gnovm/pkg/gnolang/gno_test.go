package gnolang

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"text/template"
	"unsafe"

	// "github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMachine(b *testing.B, numValues, numStmts, numExprs, numBlocks, numFrames, numExceptions int) *Machine {
	b.Helper()

	m := &Machine{
		Ops:        make([]Op, 100),
		NumOps:     100,
		Values:     make([]TypedValue, numValues),
		NumValues:  numValues,
		Exprs:      make([]Expr, numExprs),
		Stmts:      make([]Stmt, numStmts),
		Blocks:     make([]*Block, numBlocks),
		Frames:     make([]*Frame, numFrames),
		Exceptions: make([]Exception, numExceptions),
	}
	return m
}

func BenchmarkStringLargeData(b *testing.B) {
	m := setupMachine(b, 10000, 5000, 5000, 2000, 3000, 1000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = m.String()
	}
}

func TestBuiltinIdentifiersShadowing(t *testing.T) {
	t.Parallel()
	tests := map[string]string{}

	uverseNames := []string{
		"iota",
		"append",
		"cap",
		"close",
		"complex",
		"copy",
		"delete",
		"len",
		"make",
		"new",
		"panic",
		"print",
		"println",
		"recover",
		"nil",
		"bool",
		"byte",
		"float32",
		"float64",
		"int",
		"int8",
		"int16",
		"int32",
		"int64",
		"rune",
		"string",
		"uint",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"typeval",
		"error",
		"true",
		"false",
	}

	for _, name := range uverseNames {
		tests[("struct builtin " + name)] = fmt.Sprintf(`
			package test

			type %v struct {}

			func main() {}
		`, name)

		tests[("var builtin " + name)] = fmt.Sprintf(`
			package test

			func main() {
				%v := 1
			}
		`, name)

		tests[("var declr builtin " + name)] = fmt.Sprintf(`
			package test

			func main() {
				var %v int
			}
		`, name)
	}

	for n, s := range tests {
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("shadowing test: `%s` should have failed but didn't\n", n)
				}
			}()

			m := NewMachine("test", nil)
			nn := MustParseFile("main.go", s)
			m.RunFiles(nn)
			m.RunMain()
		})
	}
}

// run empty main().
func TestRunEmptyMain(t *testing.T) {
	t.Parallel()

	m := NewMachine("test", nil)
	// []Stmt{} != nil, as nil means that in the source code not even the
	// brackets are present and is reserved for external (ie. native) functions.
	main := FuncD("main", nil, nil, []Stmt{})
	m.RunDeclaration(main)
	m.RunMain()
}

// run main() with a for loop.
func TestRunLoopyMain(t *testing.T) {
	t.Parallel()

	m := NewMachine("test", nil)
	c := `package test
func main() {
	for i:=0; i<1000; i++ {
		if i == -1 {
			return
		}
	}
}`
	n := MustParseFile("main.go", c)
	m.RunFiles(n)
	m.RunMain()
}

func BenchmarkPreprocessForLoop(b *testing.B) {
	m := NewMachine("test", nil)
	c := `package test
func main() {
	for i:=0; i<10000; i++ {}
}`
	n := MustParseFile("main.go", c)
	m.RunFiles(n)

	for i := 0; i < b.N; i++ {
		m.RunMain()
	}
}

func BenchmarkIfStatement(b *testing.B) {
	m := NewMachine("test", nil)
	c := `package test
func main() {
	for i:=0; i<10000; i++ {
		if i > 10 {

		}
	}
}`
	n := MustParseFile("main.go", c)
	m.RunFiles(n)

	for i := 0; i < b.N; i++ {
		m.RunMain()
	}
}

func TestDoOpEvalBaseConversion(t *testing.T) {
	m := NewMachine("test", nil)

	type testCase struct {
		input     string
		expect    string
		expectErr bool
	}

	testCases := []testCase{
		// binary
		{input: "0b101010", expect: "42", expectErr: false},
		{input: "0B101010", expect: "42", expectErr: false},
		{input: "0b111111111111111111111111111111111111111111111111111111111111111", expect: "9223372036854775807", expectErr: false},
		{input: "0b0", expect: "0", expectErr: false},
		{input: "0b000000101010", expect: "42", expectErr: false},
		{input: " 0b101010", expectErr: true},
		{input: "0b", expectErr: true},
		{input: "0bXXXX", expectErr: true},
		{input: "42b0", expectErr: true},
		// octal
		{input: "0o42", expect: "34", expectErr: false},
		{input: "0o0", expect: "0", expectErr: false},
		{input: "042", expect: "34", expectErr: false},
		{input: "0777", expect: "511", expectErr: false},
		{input: "0O0000042", expect: "34", expectErr: false},
		{input: "0777777777777777777777", expect: "9223372036854775807", expectErr: false},
		{input: "0o777777777777777777777", expect: "9223372036854775807", expectErr: false},
		{input: "048", expectErr: true},
		{input: "0o", expectErr: true},
		{input: "0oXXXX", expectErr: true},
		{input: "0OXXXX", expectErr: true},
		{input: "0o42x42", expectErr: true},
		{input: "0O42x42", expectErr: true},
		{input: "0420x42", expectErr: true},
		{input: "0o420o42", expectErr: true},
		// hex
		{input: "0x2a", expect: "42", expectErr: false},
		{input: "0X2A", expect: "42", expectErr: false},
		{input: "0x7FFFFFFFFFFFFFFF", expect: "9223372036854775807", expectErr: false},
		{input: "0x2a ", expectErr: true},
		{input: "0x", expectErr: true},
		{input: "0xXXXX", expectErr: true},
		{input: "0xGHIJ", expectErr: true},
		{input: "0x42o42", expectErr: true},
		{input: "0x2ax42", expectErr: true},
		// decimal
		{input: "42", expect: "42", expectErr: false},
		{input: "0", expect: "0", expectErr: false},
		{input: "0000000000", expect: "0", expectErr: false},
		{input: "9223372036854775807", expect: "9223372036854775807", expectErr: false},
	}

	for _, tc := range testCases {
		m.PushExpr(&BasicLitExpr{
			Kind:  INT,
			Value: tc.input,
		})

		if tc.expectErr {
			assert.Panics(t, func() { m.doOpEval() })
		} else {
			m.doOpEval()
			v := m.PopValue()
			assert.Equal(t, tc.expect, v.V.String())
		}
	}
}

func TestEval(t *testing.T) {
	t.Parallel()

	m := NewMachine("test", nil)
	c := `package test
func next(i int) int {
	return i+1
}`
	n := MustParseFile("main.go", c)
	m.RunFiles(n)
	res := m.Eval(Call("next", "1"))
	fmt.Println(res)
}

func assertOutput(t *testing.T, input string, output string) {
	t.Helper()

	buf := new(bytes.Buffer)
	m := NewMachineWithOptions(MachineOptions{
		PkgPath: "test",
		Output:  buf,
	})
	n := MustParseFile("main.go", input)
	m.RunFiles(n)
	m.RunMain()
	assert.Equal(t, output, string(buf.Bytes()))
	err := m.CheckEmpty()
	assert.Nil(t, err)
}

// ----------------------------------------
// Benchmarks

func BenchmarkPreprocess(b *testing.B) {
	pkg := &PackageNode{
		PkgName: "main",
		PkgPath: ".main",
		FileSet: nil,
	}
	pkg.InitStaticBlock(pkg, nil)
	main := FuncD("main", nil, nil, Ss(
		A("mx", ":=", "1000000"),
		For(
			A("i", ":=", "0"),
			X("i < mx"),
			Inc("i"),
		),
	))
	pn := NewPackageNode("hey", "gno.land/p/hey", nil)
	copies := make([]*FuncDecl, b.N)
	for i := 0; i < b.N; i++ {
		copies[i] = main.Copy().(*FuncDecl)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// initStaticBlocks is always performed before a Preprocess
		initStaticBlocks(nil, pn, copies[i])
		main = Preprocess(nil, pkg, copies[i]).(*FuncDecl)
	}
}

type bdataParams struct {
	N     int
	Param string
}

func BenchmarkBenchdata(b *testing.B) {
	const bdDir = "./benchdata"
	files, err := os.ReadDir(bdDir)
	require.NoError(b, err)
	for _, file := range files {
		// Read file and parse template.
		bcont, err := os.ReadFile(filepath.Join(bdDir, file.Name()))
		cont := string(bcont)
		require.NoError(b, err)
		tpl, err := template.New("").Parse(cont)
		require.NoError(b, err)

		// Determine parameters.
		const paramString = "// param: "
		var params []string
		pos := strings.Index(cont, paramString)
		if pos >= 0 {
			paramsRaw := strings.SplitN(cont[pos+len(paramString):], "\n", 2)[0]
			params = strings.Fields(paramsRaw)
		} else {
			params = []string{""}
		}

		for _, param := range params {
			name := file.Name()
			if param != "" {
				name += "_param:" + param
			}
			b.Run(name, func(b *testing.B) {
				// Gen template with N and param.
				var buf bytes.Buffer
				require.NoError(b, tpl.Execute(&buf, bdataParams{
					N:     b.N,
					Param: param,
				}))

				// Set up machine.
				m := NewMachineWithOptions(MachineOptions{
					PkgPath: "main",
					Output:  io.Discard,
				})
				n := MustParseFile("main.go", buf.String())
				m.RunFiles(n)

				b.ResetTimer()
				m.RunMain()
			})
		}
	}
}

// ----------------------------------------
// Unsorted

type Struct1 struct {
	A int
	B int
}

func TestModifyTypeAsserted(t *testing.T) {
	t.Parallel()

	x := Struct1{1, 1}
	var v interface{} = x
	x2 := v.(Struct1)
	x2.A = 2

	// only x2 is changed.
	assert.Equal(t, 1, x.A)
	assert.Equal(t, 1, v.(Struct1).A)
	assert.Equal(t, 2, x2.A)
}

type Interface1 interface {
	Foo()
}

func TestTypeConversion(t *testing.T) {
	t.Parallel()

	x := 1
	var v interface{} = x
	if _, ok := v.(Interface1); ok {
		panic("should not happen")
	}
	v = nil
	if _, ok := v.(Interface1); ok {
		panic("should not happen")
	}
	assert.Panics(t, func() {
		// this would panic.
		z := v.(Interface1)
		fmt.Println(z)
	})
}

func TestSomething(t *testing.T) {
	t.Parallel()

	type Foo struct {
		X interface{}
	}

	type Bar struct {
		X interface{}
		Y bool
	}

	fmt.Println(unsafe.Sizeof(Foo{}))                        // 16
	fmt.Println(unsafe.Sizeof(Foo{X: reflect.ValueOf(0)}.X)) // still 16? weird.
	fmt.Println(unsafe.Sizeof(reflect.ValueOf(0)))           // 24
	fmt.Println(unsafe.Sizeof(Bar{}))
}

// XXX is there a way to test in Go as well as Gno?
func TestDeferOrder(t *testing.T) {
	t.Parallel()

	a := func() func(int, int) int {
		fmt.Println("a constructed")
		return func(x int, y int) int {
			fmt.Println("a called")
			return x + y
		}
	}
	b := func() int {
		fmt.Println("b constructed")
		return 1
	}
	c := func() int {
		fmt.Println("c constructed")
		return 2
	}
	defer a()(b(), c())
	fmt.Println("post defer")

	// should print
	// a constructed
	// b constructed
	// c constructed
	// post defer
	// a called
}

// XXX is there a way to test in Go as well as Gno?
func TestCallOrder(t *testing.T) {
	t.Parallel()

	a := func() func(int, int) int {
		fmt.Println("a constructed")
		return func(x int, y int) int {
			fmt.Println("a called")
			return x + y
		}
	}
	b := func() int {
		fmt.Println("b constructed")
		return 1
	}
	c := func() int {
		fmt.Println("c constructed")
		return 2
	}
	a()(b(), c())

	// should print
	// a constructed
	// b constructed
	// c constructed
	// a called
}

// XXX is there a way to test in Go as well as Gno?
func TestBinaryShortCircuit(t *testing.T) {
	t.Parallel()

	tr := func() bool {
		fmt.Println("t called")
		return true
	}
	fa := func() bool {
		fmt.Println("f called")
		return false
	}
	if fa() && tr() {
		fmt.Println("error")
	} else {
		fmt.Println("done")
	}
}

// XXX is there a way to test in Go as well as Gno?
func TestSwitchDefine(t *testing.T) {
	t.Parallel()

	var x interface{} = 1
	switch y := x.(type) {
	case int:
		fmt.Println("int", y) // XXX
	default:
		fmt.Println("not int")
	}
}

// XXX is there a way to test in Go as well as Gno?
func TestBinaryCircuit(t *testing.T) {
	t.Parallel()

	tr := func() bool {
		fmt.Println("tr() called")
		return true
	}
	fa := func() bool {
		fmt.Println("fa() called")
		return false
	}

	fmt.Println("case 1")
	fmt.Println(fa() && tr())
	fmt.Println("case 1")
	fmt.Println(tr() || fa())

	// should print
	// case 1
	// fa() called
	// false
	// case 1
	// tr() called
	// true
}

func TestMultiAssignment(t *testing.T) {
	t.Parallel()

	buf := make([]int, 4)
	ref := func(i int) *int {
		fmt.Printf("ref(%v) called\n", i)
		return &buf[i]
	}
	val := func(i int) int {
		fmt.Printf("val(%v) called\n", i)
		return i
	}

	*ref(0), *ref(1), *ref(2), *ref(3) = val(11), val(22), val(33), val(44)

	/*
		ref(0) called
		ref(1) called
		ref(2) called
		ref(3) called
		val(11) called
		val(22) called
		val(33) called
		val(44) called
	*/
}

// XXX is there a way to test in Go as well as Gno?
func TestCallLHS(t *testing.T) {
	t.Parallel()

	x := 1
	xptr := func() *int {
		return &x
	}
	*xptr() = 2
	assert.Equal(t, 2, x)
}

// XXX is there a way to test in Go as well as Gno?
func TestCallFieldLHS(t *testing.T) {
	t.Parallel()

	type str struct {
		X int
	}
	x := str{}
	xptr := func() *str {
		return &x
	}
	y := 0
	xptr().X, y = 2, 3
	assert.Equal(t, 2, x.X)
	assert.Equal(t, 3, y)
}

// TestInvalidBuiltinArg test for invalid arguments for builtin functions
// len, cap, make
func TestInvalidBuiltinArg(t *testing.T) {
	invalidCases := []struct {
		name   string
		source string
	}{
		// Invalid len cases
		{
			name: "Invalid len(nil)",
			source: `package test
		func main() {
		 a := len(nil) // Invalid: len() does not accept nil
		 println(a)
		}`,
		},
		{
			name: "Invalid len(42)",
			source: `package test
		  func main() {
		      a := len(42) // Invalid: len() does not accept int
		      println(a)
		  }`,
		},
		{
			name: "Invalid len(func() {})",
			source: `package test
		  func main() {
		      a := len(func() {}) // Invalid: len() does not accept function types
		      println(a)
		  }`,
		},
		// Invalid cap cases
		{
			name: "Invalid cap(nil)",
			source: `package test
		func main() {
		a := cap(nil) // Invalid: 'nil' doesn't have a capacity
		println(a)
		}`,
		},
		{
			name: "Invalid cap(1)",
			source: `package test
		func main() {
		a := cap(1) // Invalid: '1' is not a slice, array, or channel
		println(a)
		}`,
		},
		{
			name: "Invalid cap(func() {})",
			source: `package test
		func main() {
		a := cap(func() {}) // Invalid: Functions don't have capacity
		println(a)
		}`,
		},
		{
			name: "Invalid cap(struct{}{})",
			source: `package test
		func main() {
		a := cap(struct{}{}) // Invalid: Structs don't have capacity
		println(a)
		}`,
		},
		{
			name: "Invalid cap('hello')",
			source: `package test
		func main() {
		a := cap("hello") // Invalid: Strings do not have capacity in this context
		println(a)
		}`,
		},
		{
			name: "Invalid cap(map[string]int{})",
			source: `package test
		func main() {
		a := cap(map[string]int{}) // Invalid: Maps don't have capacity
		println(a)
		}`,
		},
		// Invalid make cases
		{
			name: "Invalid make(int)",
			source: `package test
		func main() {
		a := make(int) // Invalid: 'make' expects a slice, map, or channel type
		println(a)
		}`,
		},
		{
			name: "Invalid make(1)",
			source: `package test
		func main() {
		a := make(1) // Invalid: 'make' requires a valid type as its first argument
		println(a)
		}`,
		},
		{
			name: "Invalid make(func() {})",
			source: `package test
		func main() {
		a := make(func() {}) // Invalid: 'make' cannot create a function type
		println(a)
		}`,
		},
		{
			name: "Invalid make([]int, 'string')",
			source: `package test
		func main() {
		a := make([]int, "string") // Invalid: The second argument must be an integer (size of slice)
		println(a)
		}`,
		},
		{
			name: "Invalid make([]int, 10, 5)",
			source: `package test
		func main() {
		a := make([]int, 10, 5) // Invalid: The second argument must be equal or higher then 10
		println(a)
		}`,
		},
		{
			name: "Invalid make([]int, 10, 'string')",
			source: `package test
		func main() {
		a := make([]int, 10, "5") // Invalid: The second argument must be int
		println(a)
		}`,
		},
		{
			name: "Invalid make([]int, 10, -5)",
			source: `package test
		func main() {
		a := make([]int, 10, -5) // Invalid: The capacity cannot be negative
		println(a)
		}`,
		},
		{
			name: "Invalid make([]int, -10, 10)",
			source: `package test
		func main() {
		a := make([]int, -10, 10) // Invalid: The size of slice cannot be negative
		println(a)
		}`,
		},
		{
			name: "Invalid make([]int, -1)",
			source: `package test
		func main() {
		a := make([]int, -1) // Invalid: The second argument cannot be negative for slice size
		println(a)
		}`,
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			testFunc := func() {
				m := NewMachine("test", nil)

				n := MustParseFile("main.go", tc.source)
				m.RunFiles(n)
				m.RunMain()
			}

			assert.Panics(t, testFunc, "The code did not panic")
		})
	}
}

// TestValidBuiltinArg test for invalid arguments for builtin functions
// len, cap, make
func TestValidBuiltinArg(t *testing.T) {
	// t.Parallel()
	validCases := []struct {
		name   string
		source string
	}{
		// Valid len cases
		{
			name: "Valid len([]int{1, 2, 3})",
			source: `package test
		func main() {
		   a := len([]int{1, 2, 3}) // Valid: len() works with slices
		   println(a)
		}`,
		},
		{
			name: "Valid len(\"hello\")",
			source: `package test
		func main() {
		    a := len("hello") // Valid: len() works with strings
		    println(a)
		}`,
		},
		{
			name: "Valid len([3]int{1, 2, 3})",
			source: `package test
		func main() {
		    a := len([3]int{1, 2, 3}) // Valid: len() works with arrays
		    println(a)
		}`,
		},
		// Valid cap cases
		{
			name: "Valid cap([]int{1, 2, 3})",
			source: `package test
		func main() {
		a := cap([]int{1, 2, 3}) // Valid: cap() works with slices
		println(a)
		}`,
		},
		{
			name: "Valid cap(make([]int, 10))",
			source: `package test
		func main() {
		a := cap(make([]int, 10)) // Valid: cap() works with slices
		println(a)
		}`,
		},
		{
			name: "Valid cap(make([]int, 10, 10))",
			source: `package test
		func main() {
		a := cap(make([]int, 10, 10)) // Valid: cap() works with slices
		println(a)
		}`,
		},
		// Valid make cases
		{
			name: "Valid make([]int, 10)",
			source: `package test
		    func main() {
		       a := make([]int, 10) // Valid: make() works with slices
		        for _, r := range a {
		       println(r)
		}
		    }`,
		},
		{
			name: "Valid make([]int, b)",
			source: `package test
		     func main() {
				  b := 10
		        a := make([]int, b) // Valid: make() works with slices
		         for _, r := range a {
		        println(r)
		}
		     }`,
		},
		{
			name: "Valid make([]int, b, c)",
			source: `package test
		     func main() {
				  c := 10
				  b := 10
		        a := make([]int, b, c) // Valid: make() works with slices
		         for _, r := range a {
		        println(r)
		}
		     }`,
		},
		{
			name: "Valid make([]int, max(len(s), 5), capacity)",
			source: `package test
        func main() {
			s := []int{1, 2, 3}
			capacity := (len(s) * 3) / 2  // Ensures this evaluates to an integer
			a := make([]int, len(s), capacity) // Valid case
			println(len(a), cap(a)) 
		}`,
		},

		{
			name: "Valid make([]int, max(b, 5), c)",
			source: `package test
        func main() {
			s := []int{1, 2, 3}
			capacity := (len(s) * 3) / 2  // Ensure this evaluates to a valid int
			a := make([]int, len(s), capacity) // Valid case
			println(len(a), cap(a))  // Should print: 3 <valid computed cap>
		}`,
		},
		{
			name: "Valid make([]int, max(b, 5), c)",
			source: `package test
        func max(x, y int) int {
            if x > y {
                return x
            }
            return y
        }

        func main() {
            b := 3
            c := 10
            a := make([]int, max(b, 5), c) // Uses max function
            for _, r := range a {
                println(r)
            }
        }`,
		},

		{
			name: "Valid make(map[string]*T)",
			source: `package test

func main() {
	type T struct {
		Name string
	}

	var m = make(map[string]*T)
	println(m)
}`,
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMachine("test", nil)

			n := MustParseFile("main.go", tc.source)
			m.RunFiles(n)
			m.RunMain()
		})
	}
}
