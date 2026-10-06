//go:build !renvo

package rtg

import (
	"strings"
	"testing"
)

func TestInlineConstraintMatchingAndEarlyClobber(t *testing.T) {
	v := FrontendOperations(frontendTestResolved(t, frontendTestDefinition(t)), "msdos/8086")
	for _, matching := range []string{"0", "[out]"} {
		plan := BindInlineOperands(v, []InlineOperand{{Name: "out", Constraint: "=&r"}}, []InlineOperand{{Constraint: matching}, {Name: "rhs", Constraint: "c"}}, []string{"cc", "memory"}, "inline.c")
		if !plan.Ok || plan.Bindings[0].Register != plan.Bindings[1].Register || plan.Bindings[0].Register == plan.Bindings[2].Register || !plan.Memory || len(plan.Resources) != 1 || plan.Resources[0] != "flags" {
			t.Fatalf("matching %s: %+v", matching, plan)
		}
		text, ds := ExpandInlineTemplate(v, plan, "att", []byte("addw %[rhs],%[out]; jmp %l3; .L%=:"), []InlineLabel{{Name: "done", Symbol: ".Lexit17"}}, 17, "inline.c")
		if len(ds) != 0 || !strings.Contains(string(text), "jmp .Lexit17; .L17:") || !strings.Contains(string(text), "%cx") {
			t.Fatalf("template: %s %+v", text, ds)
		}
		named, ds := ExpandInlineTemplate(v, plan, "att", []byte("jmp %l[done]"), []InlineLabel{{Name: "done", Symbol: ".Lexit17"}}, 17, "inline.c")
		if len(ds) != 0 || string(named) != "jmp .Lexit17" {
			t.Fatalf("named goto: %s %+v", named, ds)
		}
	}
	readwrite := BindInlineOperands(v, []InlineOperand{{Name: "sum", Constraint: "+r"}}, []InlineOperand{{Constraint: "a"}}, nil, "inline.c")
	if !readwrite.Ok || readwrite.Bindings[0].Register == readwrite.Bindings[1].Register {
		t.Fatalf("read/write overlap: %+v", readwrite)
	}
	text, ds := ExpandInlineTemplate(v, readwrite, "att", []byte("jmp %l3"), []InlineLabel{{Name: "done", Symbol: ".Lexit"}}, 0, "inline.c")
	if len(ds) != 0 || string(text) != "jmp .Lexit" {
		t.Fatalf("implicit input numbering: %s %+v", text, ds)
	}
	impossible := BindInlineOperands(v, []InlineOperand{{Constraint: "=&a"}}, []InlineOperand{{Constraint: "a"}}, nil, "bad.c")
	if impossible.Ok {
		t.Fatal("early-clobber overlaps fixed input")
	}
	permitted := BindInlineOperands(v, []InlineOperand{{Constraint: "=a"}}, []InlineOperand{{Constraint: "a"}}, nil, "inline.c")
	if !permitted.Ok {
		t.Fatal("ordinary write-only overlap rejected", permitted.Diagnostics)
	}
}

func TestInlineConstraintImmediateMemoryAndFailures(t *testing.T) {
	v := FrontendOperations(frontendTestResolved(t, frontendTestDefinition(t)), "msdos/8086")
	plan := BindInlineOperands(v, nil, []InlineOperand{{Name: "n", Constraint: "i", Constant: true, Value: TargetOperand{Kind: "int", Value: "0x2a"}}}, nil, "inline.c")
	text, ds := ExpandInlineTemplate(v, plan, "att", []byte("movw %[n],%%ax; .size mark,%c0"), nil, 0, "inline.c")
	if !plan.Ok || len(ds) != 0 || string(text) != "movw $42,%ax; .size mark,42" {
		t.Fatalf("immediate: %s %+v %+v", text, ds, plan)
	}
	for _, test := range []struct {
		outputs, inputs []InlineOperand
		clobbers        []string
	}{
		{outputs: []InlineOperand{{Constraint: "r"}}},
		{outputs: []InlineOperand{{Constraint: "=r"}}, inputs: []InlineOperand{{Constraint: "1"}}},
		{inputs: []InlineOperand{{Constraint: "[missing]"}}},
		{outputs: []InlineOperand{{Constraint: "+r"}}, inputs: []InlineOperand{{Constraint: "0"}}},
		{inputs: []InlineOperand{{Constraint: "i"}}},
		{inputs: []InlineOperand{{Constraint: "r,m"}}},
		{outputs: []InlineOperand{{Constraint: "=a"}}, clobbers: []string{"ax"}},
		{clobbers: []string{"bogus"}},
		{clobbers: []string{"cc", "flags"}},
		{outputs: []InlineOperand{{Name: "x", Constraint: "=r"}}, inputs: []InlineOperand{{Name: "x", Constraint: "r"}}},
	} {
		bad := BindInlineOperands(v, test.outputs, test.inputs, test.clobbers, "bad.c")
		if bad.Ok || len(bad.Diagnostics) == 0 || bad.Diagnostics[0].Filename != "bad.c" {
			t.Fatalf("accepted invalid constraints: %+v", test)
		}
	}
	for _, template := range []string{"%", "%1", "%q0", "%l0", "%[missing]", "%[n"} {
		if _, ds := ExpandInlineTemplate(v, plan, "att", []byte(template), nil, 0, "bad.c"); len(ds) == 0 {
			t.Fatalf("accepted %q", template)
		}
	}
	forged := InlineConstraintPlan{Ok: true, Bindings: []InlineBinding{{Kind: "register", Register: "ax; ret"}}}
	if _, ds := ExpandInlineTemplate(v, forged, "att", []byte("%0"), nil, 0, "bad.c"); len(ds) == 0 {
		t.Fatal("interpolated forged register source")
	}
	v.Constraints = append(v.Constraints, TargetConstraint{Name: "m", Kind: "memory", Registers: []string{"si"}})
	memory := BindInlineOperands(v, []InlineOperand{{Constraint: "=m"}}, nil, nil, "inline.c")
	for _, test := range []struct{ syntax, expected string }{{"att", "(%si)"}, {"intel", "[si]"}} {
		text, ds := ExpandInlineTemplate(v, memory, test.syntax, []byte("%0"), nil, 0, "inline.c")
		if len(ds) != 0 || string(text) != test.expected {
			t.Fatalf("memory: %s %+v", text, ds)
		}
	}
}
