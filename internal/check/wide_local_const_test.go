package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestWideLocalConstants(t *testing.T) {
	for _, test := range []struct {
		source string
		valid  bool
	}{
		{"func main(){const n=256;_=append([]byte{},n)}", false},
		{"func main(){const n=1<<100;_=append([]byte{},n)}", false},
		{"func main(){const n=1<<100;const m=n-n+255;_=append([]byte{},m)}", true},
		{"func main(){const n=1<<100;const m=n-n+256;_=append([]byte{},m)}", false},
		{"func main(){const n=255;_=append([]byte{},n)}", true},
		{"func main(){const(a=iota+255;b);_=append([]byte{},b)}", false},
		{"func main(){const(a,b=iota+254,iota+255;c,d);_=append([]byte{},c)}", true},
		{"func main(){const(a,b=iota+254,iota+255;c,d);_=append([]byte{},d)}", false},
		{"func main(){const(_=iota;a;b);_=append([]byte{},b)}", true},
		{"func main(){const(_=iota+254;a;b);_=append([]byte{},b)}", false},
		{"func main(){const(a=1<<100;b);_=append([]byte{},b)}", false},
		{"func main(){const(a=iota+255;b);const c=iota;_=append([]byte{},c)}", true},
		{"const n=256;func main(){const n=n-1;_=append([]byte{},n)}", true},
		{"const n=1;const m=n;func main(){const n=256;_=append([]byte{},m)}", true},
		{"func main(){const n=255;const m=n;{const n=256;_=append([]byte{},m)}}", true},
		{"func main(){const n=256;{n:=byte(1);_=append([]byte{},n)}}", true},
		{"func main(){const n=1;{const n=256;_=append([]byte{},n)}}", false},
		{"func main(){const n=1;{const n=256;_=n};_=append([]byte{},n)}", true},
		{"func main(){const iota=256;const n=iota;_=append([]byte{},n)}", false},
		{"func main(){const n int=1;_=append([]byte{},n)}", false},
		{"func main(){const n byte=1;_=append([]byte{},n)}", true},
		{"func main(){const(n byte=1;m);_=append([]byte{},m)}", true},
		{"func main(){const(n int=1;m);_=append([]byte{},m)}", false},
		{"func main(){const n=\"x\";_=append([]byte{},n)}", false},
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + test.source)}})
		result := CheckGraphCore(graph)
		if result.Ok != test.valid || !test.valid && result.Error != CheckErrBuiltinOperand {
			t.Errorf("%s: ok=%v error=%d", test.source, result.Ok, result.Error)
		}
	}
}
