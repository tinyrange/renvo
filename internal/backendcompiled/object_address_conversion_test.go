//go:build !renvo

package backendcompiled

import "testing"

func TestObjectAddressConversionsUseAddressWidth(t *testing.T) {
	for _, tc := range []struct {
		name, typ, expression string
		want                  bool
	}{
		{"data-address", "uint64", "uint64(&object)", true},
		{"narrow-data-address", "uint32", "uint32(&object)", false},
		{"function-address", "uint64", "uint64(foreign)", true},
		{"narrow-function-address", "uint32", "uint32(foreign)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
			// Deliberately separate the language carrier from both address widths.
			context.renvoNativeIntSize = 4
			context.objectFile = true
			source := []byte("package main\nvar object int\n// renvo:linkstatic test,foreign_symbol\nfunc foreign() {}\nvar address " + tc.typ + " = " + tc.expression + "\nfunc appMain() int { return 0 }\n")
			program := renvoParseProgramWithContext(source, context)
			meta := renvoBuildMeta(&program)
			g := renvoLinearGen{c: context, prog: &program, meta: &meta}
			g.asm.c = context
			found := false
			for i := range meta.globals {
				symbol := &meta.globals[i]
				if !renvoBytesEqualText(source, symbol.nameStart, symbol.nameEnd, "address") {
					continue
				}
				found = true
				ep := renvoNewExprParse()
				root := renvoParseExpressionRoot(ep, &program, symbol.initStart, symbol.initEnd)
				if root < 0 {
					t.Fatal("initializer parse failed")
				}
				data := make([]byte, renvoTypeSize(&meta, symbol.typ))
				ok := renvoObjectStoreConstant(&g, ep, root, symbol.typ, data, 0, 24)
				if ok != tc.want {
					t.Fatalf("initializer accepted = %v, want %v", ok, tc.want)
				}
				if tc.want {
					if len(g.asm.objectDataRelocs) != 1 || g.asm.objectDataRelocs[0].offset != 24 {
						t.Fatalf("address relocation missing: %+v", g.asm.objectDataRelocs)
					}
				} else if len(g.asm.objectDataRelocs) != 0 {
					t.Fatal("narrow conversion admitted a full-width relocation")
				}
			}
			if !found {
				t.Fatal("parsed initializer missing")
			}
		})
	}
}
