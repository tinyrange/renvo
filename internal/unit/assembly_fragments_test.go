package unit

import "testing"

func TestAssemblyFragmentsRejectMalformedInterchange(t *testing.T) {
	data, ok := MarshalCore(CoreProgram{RTGAssembly: []RTGAssemblySource{{Path: "x.s", Source: []byte(".globl x\nx: ret\n")}}, RTGAssemblyFuncs: []RTGAssemblyBinding{{Func: 0, Source: 0, Entry: 0}}})
	if !ok {
		t.Fatal("marshal")
	}
	if _, _, ok := ReadRTGAssemblyFragments(data); !ok {
		t.Fatal("valid unresolved table rejected")
	}
	sources, bindings, _ := ReadRTGAssemblyFragments(data)
	resolved, ok := AttachRTGAssemblyFragments(data, bindings, [][]byte{{1, 2, 3}})
	if !ok {
		t.Fatal("attach")
	}
	after, code, ok := ReadRTGAssemblyFragments(resolved)
	if !ok || string(after[0].Source) != string(sources[0].Source) || len(code[0].Code) != 3 {
		t.Fatal("lost source or code")
	}
	damaged := append([]byte(nil), data...)
	damaged[4]++
	if _, _, ok := ReadRTGAssemblyFragments(damaged); ok {
		t.Fatal("accepted incompatible unit")
	}
	// An otherwise valid first assembly child must not hide a duplicate or
	// malformed trailing child. Validate the complete enclosing stream.
	duplicate := append([]byte(nil), data...)
	for at := 14; at+6 <= len(data); {
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		if tag == TagRTGAssembly {
			duplicate = append(duplicate, data[at:at+6+size]...)
			break
		}
		at += 6 + size
	}
	writeRTGASMUint32(duplicate, 10, len(duplicate)-14)
	if _, _, ok := ReadRTGAssemblyFragments(duplicate); ok {
		t.Fatal("accepted duplicate table")
	}
	trailing := append(append([]byte(nil), data...), 33)
	writeRTGASMUint32(trailing, 10, len(trailing)-14)
	if _, _, ok := ReadRTGAssemblyFragments(trailing); ok {
		t.Fatal("accepted trailing partial child")
	}
	// A count that wraps to zero on a 32-bit self-host must also be rejected
	// on the host, rather than selecting an alternate payload version.
	overflow := appendRTGASMNode(append([]byte(nil), data[:14]...), TagRTGAssembly, []byte{128, 128, 128, 128, 16, 2, 0, 0})
	writeRTGASMUint32(overflow, 10, len(overflow)-14)
	if _, _, ok := ReadRTGAssemblyFragments(overflow); ok {
		t.Fatal("accepted overflowing portable varint")
	}
}
