package rtg

// assemblyEvaluatorKernel projects the ordinary emitter API by reachability.
// It deliberately does not pull the whole compiler, target runtime or image
// writer into an encoder evaluator. The same emitter rewriting and definition-
// owned label finalizer used by prepared backends remain authoritative.
func assemblyEvaluatorKernel(body []byte) []byte {
	api := appendArchitectureBackendAPI([]byte(assemblyEvaluatorStorage))
	tokens, diagnostics := scan(body, "")
	if len(diagnostics) != 0 {
		return nil
	}
	var roots []string
	for i := range tokens {
		if tokens[i].Kind == TokenIdent {
			name := tokenText(body, tokens[i])
			if stringIndex(roots, name) < 0 {
				roots = append(roots, name)
			}
		}
	}
	document := Document{Declarations: []Declaration{{Kind: DeclGo, Name: "backend", GoSource: api}}}
	parts := reachableEmbeddedGoParts(document, roots, nil)
	var out []byte
	for i := range parts {
		out = append(out, parts[i].source...)
		out = append(out, '\n')
	}
	return out
}

const assemblyEvaluatorStorage = `
// The evaluator owns byte buffers and labels, not a compiler context. Opcode
// encoders and relocation algorithms are projected from the selected definition.
type renvoAsm struct {
    code, data []byte
    labelPos, relocs, absRelocs []int32
    dataOffset, bssSize, lastPrimaryLoad, signedCompareLabel int
    patchFailed bool
}
var renvoRTGUnsupportedOperation int
func renvoAsmAssemblyReady(a *renvoAsm) bool {
    if a.patchFailed || len(a.absRelocs)!=0 || len(a.data)!=0 || a.bssSize!=0 { return false }
    for i:=0; i+1<len(a.relocs); i+=2 {
        if renvoAsmLabelPosition(a,int(a.relocs[i+1]))<0 { return false }
    }
    return true
}
func renvoAsmEmit8(a *renvoAsm, v int) { a.code = append(a.code, byte(v)) }
func renvoAsmEmit16(a *renvoAsm, v int) { a.code = append(a.code, byte(v), byte(v>>8)) }
func renvoAsmEmit24(a *renvoAsm, v int) { renvoAsmEmit16(a,v); renvoAsmEmit8(a,v>>16) }
func renvoAsmEmit32(a *renvoAsm, v int) { a.code = append(a.code,byte(v),byte(v>>8),byte(v>>16),byte(v>>24)) }
func renvoAsmEmit64(a *renvoAsm, v int) { a.code = append(a.code,byte(v),byte(v>>8),byte(v>>16),byte(v>>24),byte(v>>32),byte(v>>40),byte(v>>48),byte(v>>56)) }
func renvoAsmEmit2(a *renvoAsm, x,y int) { renvoAsmEmit8(a,x); renvoAsmEmit8(a,y) }
func renvoAsmEmit3(a *renvoAsm, x,y,z int) { renvoAsmEmit2(a,x,y); renvoAsmEmit8(a,z) }
func renvoAsmEmit4(a *renvoAsm, x,y,z,w int) { renvoAsmEmit3(a,x,y,z); renvoAsmEmit8(a,w) }
func renvoAsmEmitText(a *renvoAsm, s string) { a.code = append(a.code,s...) }
func renvoAsmNewLabel(a *renvoAsm) int { n:=len(a.labelPos); a.labelPos=append(a.labelPos,-1); return n }
func renvoAsmMarkLabel(a *renvoAsm, n int) { if n<0 || n>=len(a.labelPos) || a.labelPos[n]>=0 { a.patchFailed=true; return }; a.labelPos[n]=int32(len(a.code)) }
func renvoAsmLabelPosition(a *renvoAsm, n int) int { if n<0 || n>=len(a.labelPos) { return -1 }; return int(a.labelPos[n]) }
func renvoAsmAddReloc(a *renvoAsm, at,n int) { if at<0 || at>=len(a.code) || n<0 || n>=len(a.labelPos) { a.patchFailed=true; return }; a.relocs=append(a.relocs,int32(at),int32(n)) }
func renvoAsmAddAbsReloc(a *renvoAsm, at,off,kind int) { a.absRelocs=append(a.absRelocs,int32(at),int32(off),int32(kind)) }
func renvo_runtime_UnsafeInt32At(a []int32, n int) int32 { return a[n] }
func renvoPut32At(a []byte, at,v int) { a[at]=byte(v); a[at+1]=byte(v>>8); a[at+2]=byte(v>>16); a[at+3]=byte(v>>24) }
func renvoGet32At(a []byte, at int) int { return int(a[at])|int(a[at+1])<<8|int(a[at+2])<<16|int(a[at+3])<<24 }
func renvoAlignValue(n,alignment int) int { if alignment<=0 { return n }; return (n+alignment-1)/alignment*alignment }
`
