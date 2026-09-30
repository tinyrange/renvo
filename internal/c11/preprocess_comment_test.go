package c11

import (
	"strings"
	"testing"
)

func TestDirectiveMultilineBlockComment(t *testing.T) {
	src := "#define SUM (1 /* a\nb */ + 2)\n#if SUM == 3 /* x\ny */ && 1\nint correct;\n#else\n#error wrong branch\n#endif\n"
	r := Preprocess(PreprocessConfig{Path: "comment.c", Source: []byte(src)})
	if !r.Ok || !strings.Contains(string(r.Source), "int correct") {
		t.Fatalf("%+v %q", r, r.Source)
	}
	r = Preprocess(PreprocessConfig{Path: "comment.c", Source: []byte(src + "#error location\n")})
	if r.Ok || r.Line != 9 || r.Error != PreprocessErrDirective {
		t.Fatalf("%+v", r)
	}
}
