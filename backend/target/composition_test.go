package target

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestCH32V003SeparatesObjectAndBoardContracts(t *testing.T) {
	object := RV32ECILP32E()
	if err := object.Validate(); err != nil {
		t.Fatalf("standalone object target rejected: %v", err)
	}
	composition := CH32V003()
	if err := composition.Validate(); err != nil {
		t.Fatalf("CH32V003 composition rejected: %v", err)
	}
	if !reflect.DeepEqual(composition.Object, object) {
		t.Fatalf("board changed reusable object target:\nboard:  %+v\nobject: %+v", composition.Object, object)
	}
	if composition.Board.Startup.BSS != BSSZeroedByStartup || composition.Board.Stack.Direction != StackGrowsDown {
		t.Fatalf("startup/stack contract = %+v / %+v", composition.Board.Startup, composition.Board.Stack)
	}
	if composition.Board.Runtime.Result.Transport != ResultTransportDebuggerMemory || composition.Board.Runtime.Result.Symbol != "renvores" {
		t.Fatalf("result transport = %+v", composition.Board.Runtime.Result)
	}
	if len(composition.Board.Runtime.ProvidedImports) != 0 {
		t.Fatalf("CH32V003 unexpectedly supplies hosted imports: %v", composition.Board.Runtime.ProvidedImports)
	}
}

func TestCompositionDerivesC89ProfileFromObjectContract(t *testing.T) {
	composition := CH32V003()
	profile, err := composition.C89Profile()
	if err != nil {
		t.Fatal(err)
	}
	if profile.Hosted || profile.IntBits != 32 || profile.PointerBits != 32 || profile.Endian != CEndianLittle || profile.ABI != "ilp32e" {
		t.Fatalf("C89 profile drifted from object target: %+v", profile)
	}
	preamble, err := profile.RenderC89Preamble()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte("#define RENVO_C_HOSTED 0"),
		[]byte("RENVO_C89_ASSERT(pointer_width, sizeof(void *) * CHAR_BIT == 32)"),
		[]byte("#define RENVO_C_TARGET_ABI \"ilp32e\""),
		[]byte("#define RENVO_C_RUNTIME_RESULT 1"),
	} {
		if !bytes.Contains(preamble, want) {
			t.Fatalf("derived preamble missing %q:\n%s", want, preamble)
		}
	}
}

func TestCompositionRejectsImplicitOrIncompleteFreestandingContracts(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Composition)
		want string
	}{
		{name: "hosted object", edit: func(c *Composition) { c.Object.Execution = ExecutionHosted }, want: "requires a freestanding"},
		{name: "vector symbol", edit: func(c *Composition) { c.Board.Startup.VectorSymbol = "" }, want: "vector symbol"},
		{name: "BSS", edit: func(c *Composition) { c.Board.Startup.BSS = "" }, want: "BSS initialization"},
		{name: "stack direction", edit: func(c *Composition) { c.Board.Stack.Direction = "" }, want: "stack direction"},
		{name: "heap OOM", edit: func(c *Composition) { c.Board.Runtime.Heap.OOM = "" }, want: "OOM policy"},
		{name: "volatile widths", edit: func(c *Composition) { c.Board.Runtime.Volatile.Widths = 0 }, want: "volatile access widths"},
		{name: "volatile alignment", edit: func(c *Composition) { c.Board.Runtime.Volatile.Alignment = "" }, want: "volatile alignment"},
		{name: "result symbol", edit: func(c *Composition) { c.Board.Runtime.Result.Symbol = "" }, want: "requires a symbol"},
		{name: "unadvertised heap", edit: func(c *Composition) {
			c.Board.Runtime.Operations = removeString(c.Board.Runtime.Operations, "heap")
		}, want: "not advertised"},
		{name: "duplicate operation", edit: func(c *Composition) {
			c.Board.Runtime.Operations = append(c.Board.Runtime.Operations, c.Board.Runtime.Operations[0])
		}, want: "duplicate runtime operation"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			composition := CH32V003()
			test.edit(&composition)
			err := composition.Validate()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error = %v; want %q", err, test.want)
			}
		})
	}
}
