//go:build !renvo

package backendcompiled

import "testing"

// Feed actual parsed signatures through the shared classifier and selected
// definition. These cover mixed semantic kinds, result register encoding,
// integer overflow to the stack, and rejection without partial metadata.
func TestStaticCallShapeUsesDefinitionLimits(t *testing.T) {
	for _, tc := range []struct {
		name, declaration, options string
		words, result              int
		kinds                      []byte
		fail                       bool
	}{
		{name: "mixed", declaration: "func foreign(a int, b string, c []byte, d float32, e float64) float32 { return 0 }", words: 8, result: 10, options: ",result-float64=2", kinds: []byte{0, 1, 2, 3, 4}},
		{name: "integer-stack", declaration: "func foreign(a,b,c,d,e,f,g,h,i int) {}", words: 9, result: -1, kinds: make([]byte, 9)},
		{name: "float-overflow", declaration: "func foreign(a,b,c,d,e,f,g,h,i float64) {}", words: 9, fail: true},
		{name: "integer-overflow-mixed", declaration: "func foreign(a,b,c,d,e,f,g,h,i int, j float64) {}", words: 10, fail: true},
		{name: "result-register", declaration: "func foreign() float64 { return 0 }", options: ",result-float64=8", fail: true},
		{name: "word-count", declaration: "func foreign(a string) {}", words: 1, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := renvoNewCompileContext(renvoTargetDarwinArm64, false, false, false)
			source := []byte("package main\n// renvo:linkstatic /test/library,symbol" + tc.options + "\n" + tc.declaration + "\nfunc appMain() int { return 0 }\n")
			program := renvoParseProgramWithContext(source, context)
			meta := renvoBuildMeta(&program)
			var foreign *renvoFuncInfo
			for i := range meta.funcs {
				if meta.funcs[i].linkStatic != 0 {
					foreign = &meta.funcs[i]
				}
			}
			if foreign == nil {
				t.Fatal("parsed static function missing")
			}
			g := renvoLinearGen{c: context, prog: &program, meta: &meta}
			g.asm.c = context
			g.asm.staticCallResultFloat = -99
			ok := renvoPrepareStaticCallShape(&g, foreign, tc.words)
			if ok == tc.fail {
				t.Fatalf("shape success = %v", ok)
			}
			if tc.fail {
				if g.asm.staticCallResultFloat != -99 || g.asm.staticCallParamCount != 0 {
					t.Fatal("rejected shape mutated metadata")
				}
				return
			}
			if g.asm.staticCallResultFloat != tc.result || g.asm.staticCallParamCount != len(tc.kinds) {
				t.Fatalf("result/count = %d/%d", g.asm.staticCallResultFloat, g.asm.staticCallParamCount)
			}
			for i, want := range tc.kinds {
				if g.asm.staticCallParamKinds[i] != want {
					t.Errorf("kind %d = %d, want %d", i, g.asm.staticCallParamKinds[i], want)
				}
			}
		})
	}
}
