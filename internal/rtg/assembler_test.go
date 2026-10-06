//go:build !renvo

package rtg

import (
	"bytes"
	"strings"
	"testing"
)

func TestAssemblerDialectsUseBackendForms(t *testing.T) {
	v := FrontendOperations(frontendTestResolved(t, frontendTestDefinition(t)), "msdos/8086")
	for _, test := range []struct{ syntax, source string }{
		{"att", ".text\n.globl answer\n.type answer,@function\nanswer: movw $40,%ax; movw $2,%dx; cmpw %ax,%ax; je 1f; movw $99,%ax\n1: addw %dx,%ax; ret\n.size answer,.-answer\n"},
		{"intel", ".text\n.globl answer\nanswer: mov ax,40; mov dx,2; cmp ax,ax; je .Ldone; mov ax,99\n.Ldone: add ax,dx; ret\n"},
	} {
		t.Run(test.syntax, func(t *testing.T) {
			blocks, ds := ParseTargetAssembler(v, test.syntax, []byte(test.source), "answer.s")
			if len(ds) != 0 || len(blocks) != 1 {
				t.Fatalf("parse: %+v", ds)
			}
			encoded, ds := EncodeTargetAssembly(v, blocks, "answer.rtgasm")
			if len(ds) != 0 || !bytes.Contains(encoded, []byte("move_immediate(register(ax), int(40))")) || !bytes.Contains(encoded, []byte("add(register(ax), register(dx))")) {
				t.Fatalf("IR: %s %+v", encoded, ds)
			}
		})
	}
	// The spelling is data, not an architecture/opcode switch in the parser.
	custom := frontendTestDefinition(t)
	custom = bytes.Replace(custom, []byte("ret() = return()"), []byte("finish() = return()"), 1)
	changed := FrontendOperations(frontendTestResolved(t, custom), "msdos/8086")
	_, ds := ParseTargetInstructionBlock(changed, "att", []byte("finish"), "custom.s", "answer")
	if len(ds) != 0 {
		t.Fatalf("custom spelling: %+v", ds)
	}
	_, ds = ParseTargetInstructionBlock(changed, "att", []byte("ret"), "custom.s", "answer")
	if len(ds) == 0 {
		t.Fatal("parser retained undeclared opcode")
	}
}
func TestAssemblerLocalLabelsAndDiagnostics(t *testing.T) {
	v := FrontendOperations(frontendTestResolved(t, frontendTestDefinition(t)), "msdos/8086")
	block, ds := ParseTargetInstructionBlock(v, "att", []byte("1: movw $1,%ax; jne 1b; jmp 1f\n1: ret"), "numeric.s", "answer")
	if len(ds) != 0 {
		t.Fatalf("numeric labels: %+v", ds)
	}
	if block.Instructions[4].Operands[1].Value != "asmLabel0" || block.Instructions[5].Operands[0].Value != "asmLabel1" {
		t.Fatalf("incorrect local labels: %+v", block)
	}
	for _, source := range []string{"movw $65536,%ax", "movq $1,%ax", "movw $1,%rax", "jmp 1f", "jmp other_function", "movw $1+2,%ax", "ret; out.Byte(3)", ".byte 0xc3", "1: ret; 1: jmp 2b"} {
		_, ds := ParseTargetInstructionBlock(v, "att", []byte(source), "bad.s", "answer")
		if len(ds) == 0 || ds[0].Filename != "bad.s" {
			t.Fatalf("accepted %q", source)
		}
	}
	for _, source := range []string{".text\n.globl missing", ".data\n.globl answer\nanswer: ret", ".globl answer\nanswer: ret\n.size answer,4", ".globl answer\nanswer: ret\n.globl answer"} {
		_, ds := ParseTargetAssembler(v, "att", []byte(source), "bad.s")
		if len(ds) == 0 {
			t.Fatalf("accepted file %q", source)
		}
	}
	_, ds = ParseTargetInstructionBlock(v, "att", []byte("ret\n/* unfinished"), "bad.s", "answer")
	if len(ds) == 0 || !strings.Contains(ds[0].Message, "comment") {
		t.Fatal(ds)
	}
}
func TestAssemblerContractRejectsInvalidForms(t *testing.T) {
	base := frontendTestDefinition(t)
	for _, test := range []struct{ old, new string }{
		{"ret() = return()", "ret() = helper()"},
		{"ret() = return()", "ret() = new_label()"},
		{"style = att", "style = bogus"},
		{"move(destination, source)", "move(source, missing)"},
		{"move(destination, source)", "move(destination)"},
		{"ret() = return()", "ret() = return()\nret() = return()"},
		{"branch(eq, destination)", "branch(made_up, destination)"},
	} {
		source := bytes.Replace(base, []byte(test.old), []byte(test.new), 1)
		resolved := Resolve(ParseImports(source, "../../backends/msdos.rtg", testFilesystemImportLoader{}))
		if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-SYNTAX-001") {
			t.Fatalf("accepted invalid form %q: %+v", test.new, resolved.Diagnostics)
		}
	}
}
