package driver

import (
	"testing"
	"testing/fstest"
)

func TestStaticAssertSizeofUnaryOperand(t *testing.T) {
	for _, expression := range []string{"sizeof buf == 256", "sizeof fmt == 17", "sizeof buf >= sizeof fmt", "sizeof(n)==4", "((__typeof__(n))0 < (__typeof__(n))-1)==0", "sizeof buf >= sizeof fmt + ((((sizeof(n)*8)-(!((__typeof__(n))0 < (__typeof__(n))-1)))*146+484)/485)+(!((__typeof__(n))0 < (__typeof__(n))-1))"} {
		t.Run(expression, func(t *testing.T) {
			source := "int probe(int n) {static char buf[256]; static char const fmt[]=\"Unknown error %d\"; _Static_assert(" + expression + ",\"bound\");return 0;}"
			f := memorySourceFS{files: fstest.MapFS{"probe.c": {Data: []byte(source)}}}
			r, e := CompileCommand(&CommandRequest{Filesystem: f, Args: []string{"cc", "-c", "probe.c"}, Target: "linux/amd64"})
			if e != nil {
				t.Fatal(e)
			}
			if !r.Ok {
				t.Fatalf("%+v", r.Diagnostic)
			}
		})
	}
}
