//go:build !renvo

package backendcompiled

import (
	"bytes"
	"os"
	"testing"
)

// An ILP32 carrier must not narrow a native 64-bit field load. Exercise the
// parsed selector and actual instruction emission, not just layout sizes.
func TestNativeScalarAddressAccessIndependentOfIntegerCarrier(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.renvoNativeIntSize = 4
	context.objectFile = true
	source, err := os.ReadFile("../../backend/tests/native_address_scalar_access.go")
	if err != nil {
		t.Fatal(err)
	}
	program := renvoParseProgramWithContext(source, context)
	meta := renvoBuildMeta(&program)
	for _, name := range []string{"loaded", "function"} {
		t.Run(name, func(t *testing.T) {
			g := renvoLinearGen{c: context, prog: &program, meta: &meta}
			g.asm.c = context
			for i, symbol := range meta.globals {
				g.globals = append(g.globals, renvoGlobalInfo{nameStart: symbol.nameStart, nameEnd: symbol.nameEnd, offset: i * 32})
			}
			found := false
			for i := range meta.globals {
				symbol := &meta.globals[i]
				if !renvoBytesEqualText(source, symbol.nameStart, symbol.nameEnd, name) {
					continue
				}
				found = true
				ep := renvoNewExprParse()
				root := renvoParseExpressionRoot(ep, &program, symbol.initStart, symbol.initEnd)
				if root < 0 || !renvoEmitScalarSelectorExpr(&g, ep, root) || g.asm.patchFailed {
					t.Fatal("parsed native selector could not be emitted")
				}
				// MOV RAX,[RDX], not MOV EAX,[RDX]: preserve the high address bits.
				if !bytes.HasSuffix(g.asm.code, []byte{0x48, 0x8b, 0x02}) {
					t.Fatalf("native address load lost its width: %x", g.asm.code)
				}
				if name == "function" {
					stored := false
					for _, slot := range meta.globals {
						if !renvoBytesEqualText(source, slot.nameStart, slot.nameEnd, "slot") {
							continue
						}
						left := renvoNewExprParse()
						address := renvoParseExpressionRoot(left, &program, slot.initStart, slot.initEnd)
						g.asm.code = nil
						if address < 0 || !renvoEmitPointerAssignment(&g, left, address, ep, root, symbol.typ, slot.initStart-1) || g.asm.patchFailed {
							t.Fatal("parsed function-address store could not be emitted")
						}
						if !bytes.HasSuffix(g.asm.code, []byte{0x48, 0x89, 0x02}) {
							t.Fatalf("native function-address store lost its width: %x", g.asm.code)
						}
						stored = true
					}
					if !stored {
						t.Fatal("function-address slot missing")
					}
				}
			}
			if !found {
				t.Fatal("initializer missing")
			}
		})
	}
}
