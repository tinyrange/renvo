package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestAppendIntegerConstantBounds(t *testing.T) {
	for _, test := range []struct{ typ, minimum, maximum, below, above string }{
		{"int8", "-128", "127", "-129", "128"},
		{"uint8", "0", "255", "-1", "256"},
		{"int16", "-32768", "32767", "-32769", "32768"},
		{"uint16", "0", "65535", "-1", "65536"},
		{"int32", "-2147483648", "2147483647", "-2147483649", "2147483648"},
		{"uint32", "0", "4294967295", "-1", "4294967296"},
		{"int64", "-9223372036854775808", "9223372036854775807", "-9223372036854775809", "9223372036854775808"},
		{"uint64", "0", "18446744073709551615", "-1", "18446744073709551616"},
	} {
		for i, value := range []string{test.minimum, test.maximum, test.below, test.above} {
			source := "package main\nfunc main(){_=append([]" + test.typ + "{}," + value + ")}"
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(source)}})
			result := CheckGraphCore(graph)
			if i < 2 && !result.Ok || i >= 2 && (result.Ok || result.Error != CheckErrBuiltinOperand) {
				t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
			}
		}
	}
}

func TestAppendWideConstantExpressions(t *testing.T) {
	for _, test := range []struct {
		source string
		valid  bool
	}{
		{"func main(){_=append([]byte{},(1<<100)-(1<<100)+255)}", true},
		{"func main(){_=append([]byte{},(1<<100)-(1<<100)+256)}", false},
		{"const N=1<<100;func main(){_=append([]uint64{},N)}", false},
		{"const N=1<<100;func main(){N:=1;_=append([]int{},N)}", true},
		{"type Small int8;func main(){_=append([]Small{},128)}", false},
		{"type Small int8;type Alias=Small;func main(){_=append([]Alias{},-128)}", true},
		{"func main(){_=append([]int{},1<<100)}", false},
		{"func main(){_=append([]uintptr{},-1)}", false},
		{"func main(){_=append([]byte{},(1<<100)/(1<<92)-1)}", true},
		{"func main(){_=append([]byte{},(1<<100)/(1<<92))}", false},
		{"func main(){_=append([]int8{},-257%129)}", true},
		{"func main(){_=append([]uint8{},-257%129)}", false},
		{"func main(){_=append([]int8{},(1<<100)%257)}", true},
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + test.source)}})
		result := CheckGraphCore(graph)
		if result.Ok != test.valid || !test.valid && result.Error != CheckErrBuiltinOperand {
			t.Fatalf("%s: ok=%v error=%d", test.source, result.Ok, result.Error)
		}
	}
}
