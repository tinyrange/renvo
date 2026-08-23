package starlark

import (
	"strings"
	"testing"
)

func TestSyntaxExpressions(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "precedence and arithmetic",
			source: `result = [
    2 + 3 * 4,
    (2 + 3) * 4,
    17 // 5,
    17 % 5,
			2 ** 5,
			2 ** 3 ** 2,
			-2 ** 2,
			2 ** -2,
    1 | 2 & 6 ^ 1,
    ~1,
]
`,
			want: `[14, 20, 3, 2, 32, 512, -4, 0.25, 3, -2]`,
		},
		{
			name: "float arithmetic",
			source: `result = [
    1 / 4,
    1.5 + 2.25,
    5.0 - 1.5,
    1.5 * 2.0,
    7.5 / 2.5,
    2.0 ** 3,
    3.5 > 3.25,
]
`,
			want: `[0.25, 3.75, 3.5, 3.0, 3.0, 8.0, True]`,
		},
		{
			name: "boolean and membership",
			source: `result = [
    False and missing,
    True or missing,
    not False,
    2 in [1, 2, 3],
    4 not in (1, 2, 3),
    "en" in "renvo",
    "key" in {"key": 1},
]
`,
			want: `[False, True, True, True, True, True, True]`,
		},
		{
			name: "conditional expression",
			source: `result = [
    "yes" if 1 < 2 else "no",
    1 if False else 2 if True else 3,
]
`,
			want: `["yes", 2]`,
		},
		{
			name: "collection literals",
			source: `result = [
    [],
    (),
    (1,),
    [1, 2,],
    (1, 2,),
    {"a": 1, "b": 2,},
]
`,
			want: `[[], (), (1,), [1, 2], (1, 2), {"a": 1, "b": 2}]`,
		},
		{
			name: "sequence operations",
			source: `result = [
    [1, 2] + [3],
    [1, 2] * 2,
    2 * [3],
    (1, 2) + (3,),
    (4,) * 3,
    "ab" * 2,
    2 * "xy",
]
`,
			want: `[[1, 2, 3], [1, 2, 1, 2], [3, 3], (1, 2, 3), (4, 4, 4), "abab", "xyxy"]`,
		},
		{
			name: "index and slice",
			source: `values = [0, 1, 2, 3, 4]
result = [
    values[0],
    values[-1],
    values[:2],
    values[2:],
    values[-4:-1],
    (0, 1, 2, 3)[1:3],
    "renvo"[1:4],
]
`,
			want: `[0, 4, [0, 1], [2, 3, 4], [1, 2, 3], (1, 2), "env"]`,
		},
		{
			name: "strings",
			source: `plain = 'single' + "-double"
escaped = "line\n\tquote:\""
raw = r"line\n"
triple = '''first
second'''
result = [plain, escaped, raw, triple, "value=%d/%s" % (7, "ok")]
`,
			want: `["single-double", "line\n\tquote:\"", "line\\n", "first\nsecond", "value=7/ok"]`,
		},
		{
			name: "numeric separators",
			source: `result = [1_000_000, 12_345.25]
`,
			want: `[1000000, 12345.25]`,
		},
		{
			name: "postfix chaining",
			source: `record = {"items": [" zero ", " one "]}
result = record.get("items")[1].strip().upper()
`,
			want: `"ONE"`,
		},
		{
			name: "comprehensions",
			source: `pairs = [(1, 2), (3, 4), (5, 6)]
sums = [a + b for a, b in pairs if a > 1 if b < 6]
squares = {a: b * b for a, b in pairs if a != 3}
result = [sums, squares]
`,
			want: `[[7], {1: 4, 5: 36}]`,
		},
		{
			name: "parenthesized multiline expressions",
			source: `result = (
    10
    + 20
    + 12
)
values = [
    result,
    result + 1,
]
result = values
`,
			want: `[42, 43]`,
		},
		{
			name: "unicode identifiers and strings",
			source: `café = "☕"
result = café + "!"
`,
			want: `"☕!"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			globals := execute(t, test.source)
			value := globals["result"]
			if value == nil || value.String() != test.want {
				t.Fatalf("result = %v, want %s", value, test.want)
			}
		})
	}
}

func TestSyntaxStatementsAndFunctions(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "simple statements and comments",
			source: `# leading comment
a = 19; b = 23; result = a + b # trailing comment
`,
			want: "42",
		},
		{
			name: "if elif else and inline suites",
			source: `value = 2
if value == 1: result = "one"
elif value == 2: result = "two"
else: result = "other"
`,
			want: `"two"`,
		},
		{
			name: "nested blocks and pass",
			source: `result = 0
if True:
    pass
    if True:
        result = 42
`,
			want: "42",
		},
		{
			name: "for tuple target",
			source: `result = 0
for left, right in [(1, 2), (3, 4)]:
    result += left * right
`,
			want: "14",
		},
		{
			name: "while break continue",
			source: `index = 0
result = 0
while index < 8:
    index += 1
    if index % 2 == 0:
        continue
    if index > 5:
        break
    result += index
`,
			want: "9",
		},
		{
			name: "tuple unpack and indexed assignment",
			source: `left, right = 19, 23
left, right = right, left
values = [1, 2]
values[0] += 4
mapping = {"x": 3}
mapping["x"] *= 2
result = [left, right, values, mapping]
`,
			want: `[23, 19, [5, 2], {"x": 6}]`,
		},
		{
			name: "function defaults keywords and trailing commas",
			source: `def combine(a, b=2, c=3,):
    """Combine three values."""
    return a * 100 + b * 10 + c

result = [combine(1), combine(1, 4,), combine(1, c=9, b=8,)]
`,
			want: `[123, 143, 189]`,
		},
		{
			name: "bare return",
			source: `def stop(value):
    if value:
        return
    return 7

result = [stop(True), stop(False)]
`,
			want: `[None, 7]`,
		},
		{
			name: "recursion",
			source: `def factorial(n):
    if n < 2:
        return 1
    return n * factorial(n - 1)

result = factorial(6)
`,
			want: "720",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			globals := execute(t, test.source)
			value := globals["result"]
			if value == nil || value.String() != test.want {
				t.Fatalf("result = %v, want %s", value, test.want)
			}
		})
	}
}

func TestSyntaxErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"inconsistent indentation", "if True:\n    value = 1\n  value = 2\n", "inconsistent indentation"},
		{"missing suite colon", "if True\n    value = 1\n", "expected ':'"},
		{"missing indented block", "if True:\nvalue = 1\n", "expected indented block"},
		{"unterminated string", "value = \"missing\n", "unterminated string"},
		{"unterminated list", "value = [1, 2\n", "expected ']'"},
		{"malformed dictionary", "value = {\"key\" 1}\n", "expected ':' in dict"},
		{"illegal character", "value = 1 @ 2\n", "unexpected character"},
		{"required after default", "def bad(optional=1, required):\n    pass\n", "required parameter follows default"},
		{"duplicate parameter", "def bad(value, value):\n    pass\n", "duplicate parameter"},
		{"positional after keyword", "def f(a, b): return a + b\nresult = f(a=1, 2)\n", "positional argument follows keyword"},
		{"invalid assignment target", "1 = 2\n", "cannot assign"},
		{"break outside loop", "break\n", "break outside function or loop"},
		{"continue outside loop", "continue\n", "continue outside function or loop"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ExecFile(&Thread{}, "syntax.star", test.source, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}
