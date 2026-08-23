package main

import "renvo.dev/std/starlark"

func sumBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	list := args[0].(*starlark.List)
	total := 0
	for i := 0; i < list.Len(); i++ {
		total += int(list.Index(i).(starlark.Int))
	}
	return starlark.MakeInt(total), nil
}

func builtinValue(value *starlark.Builtin) starlark.Value { return value }

func main() {
	thread := &starlark.Thread{Name: "renvo-integration"}
	thread.SetMaxExecutionSteps(100000)
	predeclared := make(starlark.StringDict)
	var sumFn starlark.BuiltinFunc
	sumFn = sumBuiltin
	sum := starlark.NewBuiltin("sum", sumFn)
	predeclared["sum"] = builtinValue(sum)
	globals, err := starlark.ExecFileOptions(&starlark.FileOptions{}, thread, "integration.star", `
def triangular(n, start=0):
    return start + sum([x for x in range(n + 1)])

def syntax_code(value, offset=2, scale=3,):
    if value < 0: return 0
    elif value == 1: return (value + offset) * scale
    else: return value

values = [1, 2]
values.append(3)
answer = triangular(6)
message = "renvo".upper() + ":" + str(values[-2:])
mapping = {x: x * x for x in range(5) if x % 2 == 0}
numbered = enumerate(sorted([3, 1, 2]), 5)
parts = "one/two/three".rsplit("/", 1)
syntax_numbers = [2 + 3 * 4, 2 ** 5, 2 ** 3 ** 2, -2 ** 2, 2 ** -2, 1.5 + 2.25, 2.0 ** 3]
syntax_sequences = [(1, 2) + (3,), (4,) * 3, [0, 1, 2, 3][-3:-1]]
syntax_filters = [a + b for a, b in [(1, 2), (3, 4), (5, 6)] if a > 1 if b < 6]
syntax_strings = [r"line\n", '''first
second''', "renvo"[1:4]]
syntax_calls = [syntax_code(-1), syntax_code(1), syntax_code(value=1, scale=4, offset=5,)]
counter = 0
total = 0
while counter < 6:
    counter += 1
    if counter == 2:
        continue
    if counter == 5:
        break
    total += counter
if answer != 21 or message != "RENVO:[2, 3]":
    fail("unexpected function or method result")
if mapping != {0: 0, 2: 4, 4: 16}:
    fail("unexpected mapping")
if numbered != [(5, 1), (6, 2), (7, 3)]:
    fail("unexpected numbered values")
if parts != ["one/two", "three"]:
    fail("unexpected split result")
if syntax_numbers != [14, 32, 512, -4, 0.25, 3.75, 8.0]:
    fail("unexpected numeric syntax result")
if syntax_sequences != [(1, 2, 3), (4, 4, 4), [1, 2]]:
    fail("unexpected sequence syntax result")
if syntax_filters != [7]:
    fail("unexpected comprehension syntax result")
if syntax_strings != ["line\\n", "first\nsecond", "env"]:
    fail("unexpected string syntax result")
if syntax_calls != [0, 9, 24]:
    fail("unexpected call syntax result")
if counter != 5 or total != 8:
    fail("unexpected loop result")
`, predeclared)
	if err != nil {
		print("FAIL\n")
		return
	}
	_ = globals
	print("PASS\n")
}
