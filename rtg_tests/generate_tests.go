//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type testCase struct {
	tier  string
	group string
	name  string
	files map[string]string
}

func main() {
	root := "rtg_tests"
	must(os.RemoveAll(filepath.Join(root, "quick")))
	must(os.RemoveAll(filepath.Join(root, "extended")))

	cases := append(quickCases(), extendedCases()...)
	for _, tc := range cases {
		writeCase(root, tc)
	}
	fmt.Printf("generated %d frontend corpus cases\n", len(cases))
}

func quickCases() []testCase {
	var out []testCase
	out = append(out, quickArithmetic(40)...)
	out = append(out, quickControl(35)...)
	out = append(out, quickStringsSlices(45)...)
	out = append(out, quickStructsMethods(45)...)
	out = append(out, quickPackages(40)...)
	out = append(out, quickArrays(25)...)
	out = append(out, quickFunctions(20)...)
	return out
}

func extendedCases() []testCase {
	groups := []struct {
		name string
		fn   func(int) []testCase
	}{
		{"maps", extendedMaps},
		{"interfaces", extendedInterfaces},
		{"arrays", extendedArrays},
		{"function_values", extendedFunctionValues},
		{"closures", extendedClosures},
		{"defer_panic_recover", extendedDeferPanicRecover},
		{"package_init", extendedPackageInit},
		{"composites", extendedComposites},
		{"conversions", extendedConversions},
		{"slices", extendedSlices},
		{"strings", extendedStrings},
		{"methods", extendedMethods},
		{"unsafe", extendedUnsafe},
		{"multi_package", extendedMultiPackage},
		{"control_flow", extendedControlFlow},
	}
	var out []testCase
	for _, group := range groups {
		out = append(out, group.fn(150)...)
	}
	return out
}

func quickArithmetic(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		a := 3 + i%17
		b := 5 + (i*7)%19
		c := 2 + (i*3)%11
		want := (a+b)*c - b + a%5
		body := fmt.Sprintf(`package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%%5
	return total
}

func main() {
	if calc(%d, %d, %d) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, a, b, c, want)
		out = append(out, simpleCase("quick", "arithmetic", i, body))
	}
	return out
}

func quickControl(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		limit := 5 + i%13
		want := 0
		for j := 0; j < limit; j++ {
			if j%3 == 0 {
				want += j * 2
			} else if j%3 == 1 {
				want += j + 4
			} else {
				want -= j
			}
		}
		body := fmt.Sprintf(`package main

func score(limit int) int {
	total := 0
	for i := 0; i < limit; i++ {
		if i%%3 == 0 {
			total = total + i*2
		} else if i%%3 == 1 {
			total = total + i + 4
		} else {
			total = total - i
		}
	}
	return total
}

func main() {
	if score(%d) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, limit, want)
		out = append(out, simpleCase("quick", "control", i, body))
	}
	return out
}

func quickStringsSlices(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		prefix := strings.Repeat("x", i%5)
		suffix := strings.Repeat("y", (i/5)%5)
		wantLen := len(prefix) + 4 + len(suffix)
		body := fmt.Sprintf(`package main

func makeText() string {
	var buf []byte
	text := "%sPASS%s"
	for i := 0; i < len(text); i++ {
		buf = append(buf, text[i])
	}
	return string(buf)
}

func main() {
	text := makeText()
	start := len("%s")
	end := len(text) - len("%s")
	if text[start:end] == "PASS" && len(text) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, prefix, suffix, prefix, suffix, wantLen)
		out = append(out, simpleCase("quick", "strings_slices", i, body))
	}
	return out
}

func quickStructsMethods(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		a := i%11 + 1
		b := (i*3)%17 + 2
		want := a*10 + b + 3
		body := fmt.Sprintf(`package main

type pair struct {
	a int
	b int
}

func (p pair) score() int {
	return p.a*10 + p.b
}

func (p *pair) add(v int) {
	p.b = p.b + v
}

func main() {
	p := pair{a: %d, b: %d}
	p.add(3)
	if p.score() == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, a, b, want)
		out = append(out, simpleCase("quick", "structs_methods", i, body))
	}
	return out
}

func quickPackages(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		mod := modulePath("quick", "packages", i)
		a := i%23 + 1
		b := (i*5)%29 + 3
		main := fmt.Sprintf(`package main

import "%s/pkg/lib"

func main() {
	if lib.Score(%d) == %d {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
`, mod, b, a+b+7)
		lib := fmt.Sprintf(`package lib

const base = %d

func Score(v int) int {
	return base + v + extra()
}
`, a)
		extra := `package lib

func extra() int {
	return 7
}

func Text() string {
	return "PASS\n"
}
`
		out = append(out, moduleCase("quick", "packages", i, map[string]string{
			"cmd/app/main.go":  main,
			"pkg/lib/lib.go":   lib,
			"pkg/lib/extra.go": extra,
		}))
	}
	return out
}

func quickArrays(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		a := i%9 + 1
		b := (i*2)%9 + 2
		c := (i*3)%9 + 3
		want := a + b*2 + c*3
		body := fmt.Sprintf(`package main

func main() {
	values := [3]int{%d, %d, %d}
	total := values[0] + values[1]*2 + values[2]*3
	if total == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, a, b, c, want)
		out = append(out, simpleCase("quick", "arrays", i, body))
	}
	return out
}

func quickFunctions(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		v := 3 + i%9
		want := fib(v)
		body := fmt.Sprintf(`package main

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	if fib(%d) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, v, want)
		out = append(out, simpleCase("quick", "functions", i, body))
	}
	return out
}

func extendedMaps(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

func main() {
	m := map[string]int{"a": %d, "b": %d}
	m["a"] = m["a"] + m["b"]
	if m["a"] == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%17+1, i%13+2, i%17+1+i%13+2)
		out = append(out, simpleCase("extended", "maps", i, body))
	}
	return out
}

func extendedInterfaces(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

type scorer interface {
	score() int
}

type item struct {
	value int
}

func (i item) score() int {
	return i.value + %d
}

func check(s scorer) bool {
	return s.score() == %d
}

func main() {
	if check(item{value: %d}) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%9, i%11+3+i%9, i%11+3)
		out = append(out, simpleCase("extended", "interfaces", i, body))
	}
	return out
}

func extendedArrays(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

func main() {
	grid := [2][3]int{{1, %d, 3}, {4, 5, %d}}
	if grid[0][1]+grid[1][2] == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%10, i%7+2, i%10+i%7+2)
		out = append(out, simpleCase("extended", "arrays", i, body))
	}
	return out
}

func extendedFunctionValues(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

func add(a int, b int) int {
	return a + b
}

func mul(a int, b int) int {
	return a * b
}

func apply(fn func(int, int) int, a int, b int) int {
	return fn(a, b)
}

func main() {
	fn := add
	if %d%%2 == 1 {
		fn = mul
	}
	if apply(fn, %d, %d) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i, i%8+2, i%5+3, choose(i%2 == 1, (i%8+2)*(i%5+3), (i%8+2)+(i%5+3)))
		out = append(out, simpleCase("extended", "function_values", i, body))
	}
	return out
}

func extendedClosures(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

func makeAdder(base int) func(int) int {
	return func(v int) int {
		return base + v
	}
}

func main() {
	add := makeAdder(%d)
	if add(%d) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%17, i%19, i%17+i%19)
		out = append(out, simpleCase("extended", "closures", i, body))
	}
	return out
}

func extendedDeferPanicRecover(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

func guarded(v int) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = v == %d
		}
	}()
	if v == %d {
		panic("expected")
	}
	return false
}

func main() {
	if guarded(%d) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%23, i%23, i%23)
		out = append(out, simpleCase("extended", "defer_panic_recover", i, body))
	}
	return out
}

func extendedPackageInit(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		mod := modulePath("extended", "package_init", i)
		want := i%31 + 8
		main := fmt.Sprintf(`package main

import "%s/pkg/lib"

func main() {
	if lib.Value() == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, mod, want)
		lib := fmt.Sprintf(`package lib

var base = %d
var total = base + extra
var extra = 8

func Value() int {
	return total
}
`, i%31)
		out = append(out, moduleCase("extended", "package_init", i, map[string]string{
			"cmd/app/main.go": main,
			"pkg/lib/lib.go":  lib,
		}))
	}
	return out
}

func extendedComposites(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

type inner struct {
	a int
}

type outer struct {
	name string
	list []inner
}

func main() {
	v := outer{name: "ok", list: []inner{{a: %d}, {a: %d}}}
	if v.name == "ok" && v.list[0].a+v.list[1].a == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%17, i%19, i%17+i%19)
		out = append(out, simpleCase("extended", "composites", i, body))
	}
	return out
}

func extendedConversions(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

type count int
type text string

func main() {
	v := count(%d)
	s := text("PASS\n")
	if int(v)+len(string(s)) == %d {
		print(string(s))
		return
	}
	print("FAIL\n")
}
`, i%37, i%37+5)
		out = append(out, simpleCase("extended", "conversions", i, body))
	}
	return out
}

func extendedSlices(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

func main() {
	values := []int{%d, %d, %d}
	values = append(values[1:2], %d)
	if len(values) == 2 && values[0]+values[1] == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%11, i%13+1, i%17+2, i%19+3, i%13+1+i%19+3)
		out = append(out, simpleCase("extended", "slices", i, body))
	}
	return out
}

func extendedStrings(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

func main() {
	text := "%sPASS\n%s"
	start := len("%s")
	end := len(text) - len("%s")
	if text[start:end] == "PASS\n" {
		print(text[start:end])
		return
	}
	print("FAIL\n")
}
`, strings.Repeat("a", i%9), strings.Repeat("b", i%7), strings.Repeat("a", i%9), strings.Repeat("b", i%7))
		out = append(out, simpleCase("extended", "strings", i, body))
	}
	return out
}

func extendedMethods(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = %d
	if int(c.add(%d)) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%31, i%17, i%31+i%17)
		out = append(out, simpleCase("extended", "methods", i, body))
	}
	return out
}

func extendedUnsafe(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(`package main

import "unsafe"

type pair struct {
	a int32
	b int32
}

func main() {
	v := pair{a: %d, b: %d}
	p := unsafe.Pointer(&v)
	q := (*pair)(p)
	if int(q.a)+int(q.b) == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, i%11, i%13, i%11+i%13)
		out = append(out, simpleCase("extended", "unsafe", i, body))
	}
	return out
}

func extendedMultiPackage(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		mod := modulePath("extended", "multi_package", i)
		main := fmt.Sprintf(`package main

import "%s/pkg/a"
import "%s/pkg/b"

func main() {
	if a.Value()+b.Value() == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, mod, mod, i%19+i%23+3)
		a := fmt.Sprintf(`package a

func Value() int {
	return %d
}
`, i%19)
		b := fmt.Sprintf(`package b

import "%s/pkg/a"

func Value() int {
	return %d + a.Value() - a.Value()
}
`, mod, i%23+3)
		out = append(out, moduleCase("extended", "multi_package", i, map[string]string{
			"cmd/app/main.go": main,
			"pkg/a/a.go":      a,
			"pkg/b/b.go":      b,
		}))
	}
	return out
}

func extendedControlFlow(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		limit := 6 + i%10
		want := 0
		for j := 0; j < limit; j++ {
			if j%5 == 0 {
				continue
			}
			if j > limit-2 {
				break
			}
			want += j
		}
		body := fmt.Sprintf(`package main

func main() {
	total := 0
	for i := 0; i < %d; i++ {
		if i%%5 == 0 {
			continue
		}
		if i > %d-2 {
			break
		}
		total = total + i
	}
	if total == %d {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`, limit, limit, want)
		out = append(out, simpleCase("extended", "control_flow", i, body))
	}
	return out
}

func simpleCase(tier string, group string, index int, main string) testCase {
	return moduleCase(tier, group, index, map[string]string{"cmd/app/main.go": main})
}

func moduleCase(tier string, group string, index int, files map[string]string) testCase {
	name := fmt.Sprintf("%03d_%s", index, strings.ReplaceAll(group, "_", ""))
	return testCase{tier: tier, group: group, name: name, files: files}
}

func writeCase(root string, tc testCase) {
	dir := filepath.Join(root, tc.tier, tc.group, tc.name)
	must(os.MkdirAll(dir, 0755))
	mod := modulePath(tc.tier, tc.group, caseIndex(tc.name))
	must(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+mod+"\n\ngo 1.25\n"), 0644))
	for name, content := range tc.files {
		path := filepath.Join(dir, name)
		must(os.MkdirAll(filepath.Dir(path), 0755))
		must(os.WriteFile(path, []byte(content), 0644))
	}
}

func modulePath(tier string, group string, index int) string {
	group = strings.ReplaceAll(group, "_", "")
	return fmt.Sprintf("example.com/rtgtests/%s/%s/case%03d", tier, group, index)
}

func caseIndex(name string) int {
	var n int
	_, err := fmt.Sscanf(name, "%03d_", &n)
	if err != nil {
		panic(err)
	}
	return n
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func choose(cond bool, a int, b int) int {
	if cond {
		return a
	}
	return b
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
