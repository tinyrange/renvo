package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestIncrementalLoweringMatchesWholeProgram(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"integer_range", `func main() { sum := 0; for i := range 4 { sum += i }; print(sum) }`},
		{"global_closure", `var twice = func(n int) int { return n * 2 }; func main() { print(twice(3)) }`},
		{"captured_closure", `func main() { n := 3; next := func() int { n++; return n }; print(next(), next()) }`},
		{"anonymous_type", `func main() { value := struct { N int }{N: 4}; print(value.N) }`},
		{"interface_method_expression", `type Reader interface { Read() int }; type Value struct {}; func (Value) Read() int { return 7 }; func main() { var v Reader = Value{}; read := Reader.Read; print(read(v)) }`},
		{"ordinary_builtins", `func main() { a := []int{1,2}; clear(a); print(min(max(2,3),4), string('雪'), a[0]) }`},
		{"unicode_identifiers", `func 五() int { return 5 }; func main() { print(五()) }`},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.23\n")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + test.source + "\n")},
			}
			want := LinkBuildCore(buildFromFiles(t, files))
			if !want.Ok {
				t.Fatalf("whole-program link failed: %d", want.Error)
			}
			for _, transient := range []bool{false, true} {
				session := BeginPackageSession(buildFromFiles(t, files), transient)
				for !session.Step() {
				}
				got := session.Result()
				if !got.Ok || !bytes.Equal(got.Data, want.Data) {
					t.Fatalf("incremental result differs (transient=%v, ok=%v):\nwhole:\n%s\nincremental:\n%s", transient, got.Ok, want.Program.Text, got.Program.Text)
				}
			}
		})
	}
}
