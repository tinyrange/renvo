package starlark

import (
	"strings"
	"testing"
)

func execute(t *testing.T, source string) StringDict {
	t.Helper()
	thread := &Thread{Name: "test"}
	thread.SetMaxExecutionSteps(100_000)
	got, err := ExecFileOptions(&FileOptions{}, thread, "test.star", source, nil)
	if err != nil {
		t.Fatalf("ExecFileOptions: %v", err)
	}
	return got
}

func TestLanguageCore(t *testing.T) {
	got := execute(t, `
def total(values, start=0):
    """Add values."""
    result = start
    for value in values:
        if value < 0:
            continue
        result += value
    return result

pairs = [(1, 2), (3, 4)]
sums = [a + b for a, b in pairs if a < 3]
answer = total(sums, start=10)
text = ",".join([str(x) for x in sums])
`)
	if got["answer"].String() != "13" {
		t.Fatalf("answer = %s", got["answer"])
	}
	if got["text"].String() != `"3"` {
		t.Fatalf("text = %s", got["text"])
	}
	fn, ok := got["total"].(*Function)
	if !ok || fn.Doc() != "Add values." || fn.ParamDefault(1).String() != "0" {
		t.Fatalf("function metadata: %#v", got["total"])
	}
}

func TestFailBuiltinReportsConfiguredError(t *testing.T) {
	thread := &Thread{Name: "test"}
	_, err := ExecFileOptions(&FileOptions{}, thread, "test.star", `fail("focused tests only")`, nil)
	if err == nil || !strings.Contains(err.Error(), "focused tests only") {
		t.Fatalf("fail error = %v", err)
	}
}

func TestCollectionsSlicesAndMethods(t *testing.T) {
	got := execute(t, `
values = [1, 2]
values.append(3)
mapping = {"name": "staragent"}
result = mapping.get("name").upper() + str(values[-2:])
membership = 4 not in values and not 1 == 2
`)
	if got["result"].String() != `"STARAGENT[2, 3]"` {
		t.Fatal(got["result"])
	}
	if got["membership"].String() != "True" {
		t.Fatal(got["membership"])
	}
}

func TestAdditionalLanguageFeatures(t *testing.T) {
	got := execute(t, `
left = "alpha"
right = "beta"
left, right
pair = left, right
swapped_left, swapped_right = right, left

pattern = r'^\s+"quoted"$'
raw_quote = r'can\'t'
program = r'''first\nsecond'''
mapping = {word: len(word) for word in ["a", "bb", "ccc"] if word != "bb"}
path_parts = "one/two/three".rsplit("/", 1)
all_parts = "one  two\tthree".rsplit()
limited_parts = " one  two\tthree ".rsplit(None, 1)
`)
	if got["pattern"].String() != `"^\\s+\"quoted\"$"` {
		t.Fatalf("pattern = %s", got["pattern"])
	}
	if got["pair"].String() != `("alpha", "beta")` || got["swapped_left"].String() != `"beta"` || got["swapped_right"].String() != `"alpha"` {
		t.Fatalf("tuple values: pair=%s swapped=(%s, %s)", got["pair"], got["swapped_left"], got["swapped_right"])
	}
	if got["program"].String() != `"first\\nsecond"` {
		t.Fatalf("program = %s", got["program"])
	}
	if got["raw_quote"].String() != `"can't"` {
		t.Fatalf("raw_quote = %s", got["raw_quote"])
	}
	if got["mapping"].String() != `{"a": 1, "ccc": 3}` {
		t.Fatalf("mapping = %s", got["mapping"])
	}
	if got["path_parts"].String() != `["one/two", "three"]` {
		t.Fatalf("path_parts = %s", got["path_parts"])
	}
	if got["all_parts"].String() != `["one", "two", "three"]` {
		t.Fatalf("all_parts = %s", got["all_parts"])
	}
	if got["limited_parts"].String() != `[" one  two", "three"]` {
		t.Fatalf("limited_parts = %s", got["limited_parts"])
	}
}

func TestExecutionLimitStopsLoop(t *testing.T) {
	thread := &Thread{Name: "test"}
	thread.SetMaxExecutionSteps(100)
	_, err := ExecFileOptions(&FileOptions{}, thread, "loop.star", "while True: pass", nil)
	if err == nil || !strings.Contains(err.Error(), "too many steps") {
		t.Fatalf("error = %v", err)
	}
}

func TestControlFlowBuiltinsAndMutation(t *testing.T) {
	got := execute(t, `
values = [1]
values.extend((2, 3, 4))
removed = values.pop(1)
position = values.index(4)
descending = list(range(5, 0, -2))
ordered = sorted([3, 1, 2])
numbered = enumerate(["a", "b"], 4)
mapping = {"a": 1, "b": 2}
dict_view = [mapping.keys(), mapping.values(), mapping.items()]
text = "  Alpha-Beta  ".strip().lower().replace("-", ":").split(":")
total = 0
i = 0
while i < 8:
    i += 1
    if i == 2:
        continue
    if i == 5:
        break
    total += i
floor = -7 // 3
remainder = -7 % 3
chosen = "yes" if total == 8 else "no"
structural = [1, [2, 3]] == [1, [2, 3]] and {"x": [4]} == {"x": [4]}
`)
	wants := map[string]string{
		"values":     "[1, 3, 4]",
		"removed":    "2",
		"position":   "2",
		"descending": "[5, 3, 1]",
		"ordered":    "[1, 2, 3]",
		"numbered":   `[(4, "a"), (5, "b")]`,
		"dict_view":  `[["a", "b"], [1, 2], [("a", 1), ("b", 2)]]`,
		"text":       `["alpha", "beta"]`,
		"total":      "8",
		"floor":      "-3",
		"remainder":  "2",
		"chosen":     `"yes"`,
		"structural": "True",
	}
	for name, want := range wants {
		if value := got[name]; value == nil || value.String() != want {
			t.Fatalf("%s = %v, want %s", name, value, want)
		}
	}
}

func TestFunctionCallAPIAndPrintHook(t *testing.T) {
	var printed string
	thread := &Thread{Name: "call", Print: func(_ *Thread, message string) { printed = message }}
	globals, err := ExecFile(thread, "call.star", `
def greet(name, punctuation="!"):
    return "hello " + name + punctuation
print("loaded", 2)
`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if printed != "loaded 2" {
		t.Fatalf("print hook = %q", printed)
	}
	fn := globals["greet"]
	value, err := Call(thread, fn, Tuple{String("Renvo")}, []Tuple{{String("punctuation"), String("?")}})
	if err != nil || value.String() != `"hello Renvo?"` {
		t.Fatalf("Call = (%v, %v)", value, err)
	}
	if _, err := Call(thread, fn, Tuple{String("one"), String("!"), String("extra")}, nil); err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("arity error = %v", err)
	}
}

func TestSourceAndParseErrors(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"value = missing", "not defined"},
		{"def bad(optional=1, required):\n    pass\n", "required parameter follows default"},
		{"value = [1, 2", "expected ']'"},
		{"value = {[1]: 2}", "unhashable"},
		{"break", "outside function or loop"},
	}
	for _, test := range tests {
		_, err := ExecFile(&Thread{}, "error.star", test.source, nil)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("ExecFile(%q) error = %v, want %q", test.source, err, test.want)
		}
	}
	if _, err := ExecFile(&Thread{}, "error.star", 17, nil); err == nil || !strings.Contains(err.Error(), "unsupported source") {
		t.Fatalf("source type error = %v", err)
	}
}

func BenchmarkEvalArithmetic(b *testing.B) {
	source := `result = 0
for x in range(100):
    result += x
`
	for range b.N {
		thread := &Thread{}
		thread.SetMaxExecutionSteps(100_000)
		if _, err := ExecFileOptions(&FileOptions{}, thread, "bench.star", source, nil); err != nil {
			b.Fatal(err)
		}
	}
}
