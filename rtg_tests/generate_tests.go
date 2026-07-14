//go:build ignore

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type testCase struct {
	tier    string
	group   string
	name    string
	module  string
	files   map[string]string
	variant string
}

type corpusManifest struct {
	Version int                   `json:"version"`
	Groups  []corpusManifestGroup `json:"groups"`
}

type corpusManifestGroup struct {
	Tier             string         `json:"tier"`
	Group            string         `json:"group"`
	Cases            int            `json:"cases"`
	StructuralShapes int            `json:"structural_shapes"`
	Variants         map[string]int `json:"variants"`
}

func main() {
	root := "rtg_tests"
	must(os.RemoveAll(filepath.Join(root, "quick")))
	must(os.RemoveAll(filepath.Join(root, "extended")))

	cases := append(quickCases(), extendedCases()...)
	for _, tc := range cases {
		writeCase(root, tc)
	}
	manifest := buildCorpusManifest(root, cases)
	data, err := json.MarshalIndent(manifest, "", "  ")
	must(err)
	data = append(data, '\n')
	must(os.WriteFile(filepath.Join(root, "corpus_manifest.json"), data, 0644))
	fmt.Printf("generated %d frontend corpus cases across %d structural groups\n", len(cases), len(manifest.Groups))
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
	out = append(out, quickLiterals(1)...)
	out = append(out, legacyIssueRegressions()...)
	out = append(out, quickBuildConstraints()...)
	out = append(out, aarch64IdentPartRegression())
	return out
}

func aarch64IdentPartRegression() testCase {
	return testCase{
		tier:   "quick",
		group:  "regressions",
		name:   "000_aarch64_ident_part",
		module: "example.com/rtgtests/quick/regressions/aarch64identpart",
		files: map[string]string{
			"cmd/app/main.go": `package main

func identStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func identPart(c byte) bool {
	return identStart(c) || (c >= '0' && c <= '9')
}

func scanIdent(src []byte, pos int) int {
	if pos >= len(src) || !identStart(src[pos]) {
		return pos
	}
	pos++
	for pos < len(src) && identPart(src[pos]) {
		pos++
	}
	return pos
}

func main() {
	if scanIdent([]byte("package main"), 0) != 7 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`,
		},
	}
}

func quickBuildConstraints() []testCase {
	files := map[string]string{
		"cmd/app/main.go": `package main

func main() {
	if platformValue()+legacyValue()+modernValue() == 44 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
		"cmd/app/platform_linux_amd64.go":   "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/platform_linux_arm64.go":   "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/platform_linux_386.go":     "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/platform_linux_arm.go":     "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/platform_darwin_arm64.go":  "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/platform_windows_amd64.go": "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/platform_windows_386.go":   "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/platform_wasip1_wasm.go":   "package main\nfunc platformValue() int { return 42 }\n",
		"cmd/app/legacy_unix.go": `// +build linux darwin

package main
func legacyValue() int { return 1 }
`,
		"cmd/app/legacy_other.go": `// +build windows wasip1

package main
func legacyValue() int { return 1 }
`,
		"cmd/app/modern_unix.go": `//go:build linux || darwin

package main
func modernValue() int { return 1 }
`,
		"cmd/app/modern_other.go": `//go:build windows || wasip1

package main
func modernValue() int { return 1 }
`,
	}
	return []testCase{moduleCase("quick", "build_constraints", 0, files)}
}

// legacyIssueRegressions preserves distinct semantic reproducers collected
// from an older compiler implementation.
func legacyIssueRegressions() []testCase {
	cases := []struct {
		issue  int
		source string
	}{
		{55, `package main

import "os"

func countEntries(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return -1
	}
	return len(entries)
}

func main() {
	if countEntries(".") > 0 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`},
		{54, `package main

import "strings"

func main() {
	parts := strings.Split("abc", "")
	if len(parts) == 3 && parts[0] == "a" && parts[1] == "b" && parts[2] == "c" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`},
		{53, `package main

func pair() (int, int) { return 2, 3 }
func add(a, b int) int { return a + b }

func main() {
	if add(pair()) == 5 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{52, `package main

type inner struct{ n int }
type outer struct{ in inner }

func main() {
	o := outer{in: inner{n: 1}}
	o.in = inner{n: 7}
	if o.in.n == 7 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{51, `package main

func main() {
	n := 0
	for range []int{1, 2} { n++ }
	if n == 2 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{50, `package main

func main() {
	m := map[string]int{"ab": 7}
	a := "a"
	if m[a+"b"] == 7 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{49, `package main

func main() {
	if "abcdef"[1:4] == "bcd" { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{48, `package main

func main() {
	x := 4
	p := &x
	(*p)++
	if x == 5 && *p == 5 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{47, `package main

func main() {
	s := make([]int, 1, 1)
	s[0] = 3
	t := append(s, 4)
	if s[0] == 3 && t[0] == 3 && t[1] == 4 && len(t) == 2 && cap(t) >= 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`},
		{46, `package main

func main() {
	if 7 / -3 == -2 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{45, `package main

func main() {
	a, b := map[string]int{"x": 1}, map[string]int{"y": 2}
	if a["x"] == 1 && b["y"] == 2 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{44, `package main

type issue44S struct{ x int }
func (s issue44S) Bump() int { s.x++; return s.x }

func main() {
	s := issue44S{x: 3}
	if s.Bump() == 4 && s.x == 3 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{43, `package main

func main() {
	if "abc"[1] == byte('b') { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{42, `package main

func main() {
	if 3 * -2 == -6 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{41, `package main

func main() {
	a := "x"
	b := "xy"[0:1]
	if a == b { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{40, `package main

type issue40A struct{ n int }
type issue40B struct{ n int }
func (x issue40A) val() int { return x.n + 1 }
func (x issue40B) val() int { return x.n + 10 }

func main() {
	aa := issue40A{n: 2}
	bb := issue40B{n: 3}
	if aa.val() == 3 && bb.val() == 13 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{39, `package main

type issue39I interface{ V() int }
type issue39P struct{ n int }
func (p issue39P) V() int { return p.n }

func main() {
	p := issue39P{n: 1}
	var i issue39I = p
	p.n = 9
	if i.V() == 1 && p.n == 9 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{38, `package main

func main() {
	a, b := 1, 2
	p := &a
	*p, b = b, *p
	if a == 2 && b == 1 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{37, `package main

func main() {
	m := map[string]int{"a": 1}
	x := 2
	m["a"], x = x, m["a"]
	if m["a"] == 2 && x == 1 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{36, `package main

func main() {
	a := [2]int{1, 2}
	a[0], a[1] = 9, 8
	if a[0] == 9 && a[1] == 8 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{35, `package main

func main() {
	xs := []int{1, 2}
	ys := xs
	if len(ys) == 2 && cap(ys) == 2 && ys[0] == 1 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{34, `package main

type issue34T struct{ s string }

func main() {
	t := issue34T{s: "a"}
	t.s = "b" + "c"
	if t.s == "bc" { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{33, `package main

func main() {
	i := 9
	for i := 0; i < 3; i++ {}
	if i == 9 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{31, `package main

func main() {
	a := [0]int{}
	if len(a) == 0 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{30, `package main

func main() {
	xs := []int{1, 2}
	xs[0], xs[1] = xs[1], xs[0]
	if xs[0] == 2 && xs[1] == 1 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{29, `package main

func main() {
	total := 0
	for _, v := range [1]int{7} { total += v }
	if total == 7 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{28, `package main

func main() {
	if 7 % -4 == 3 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{27, `package main

type issue27Pair struct{ a int; b int }

func main() {
	p := issue27Pair{}
	p.a, p.b = 3, 4
	if p.a == 3 && p.b == 4 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{26, `package main

func main() {
	var m map[string]int
	m = make(map[string]int)
	m["x"] = 3
	if m["x"] == 3 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{25, `package main

type issue25S struct{ n int }

func main() {
	var a, b = issue25S{n: 1}, issue25S{n: 2}
	if a.n == 1 && b.n == 2 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{24, `package main

func issue24Pair() (int, string) { return 3, "ok" }

func main() {
	_, s := issue24Pair()
	if s == "ok" { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{23, `package main

func main() {
	k := "a"
	m := map[string]int{k: 2}
	if m["a"] == 2 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{22, `package main

import "strings"

func main() {
	parts := strings.Split("abc", ",")
	if len(parts) == 1 && parts[0] == "abc" { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{21, `package main

func main() {
	var a = [2]int{1, 2}
	if len(a) == 2 && a[0] == 1 && a[1] == 2 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{20, `package main

import "strings"

func main() {
	if strings.Count("abc", "") == 4 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{19, `package main

type issue19T struct{ N int }
func (t issue19T) Name() string { return "ok" }

func main() {
	v := issue19T{N: 1}
	if v.Name() == "ok" { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{18, `package main

type issue18T struct{ x int }
func issue18Set(t issue18T) { t.x = 9 }

func main() {
	t := issue18T{x: 3}
	issue18Set(t)
	if t.x == 3 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{17, `package main

func main() {
	m := map[string]int{"a": 1}
	got := ""
	for k := range m { got = k }
	if got == "a" { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{16, `package main

import "strings"

func main() {
	parts := strings.Split("a--b--", "--")
	if len(parts) == 3 && parts[0] == "a" && parts[1] == "b" && parts[2] == "" && strings.Join(parts, ":") == "a:b:" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`},
		{15, `package main

func main() {
	xs := []int{1, 2}
	ys := append(xs, 3)
	if len(ys) == 3 && ys[2] == 3 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{14, `package main

func main() {
	x := 1
	inner := 0
	{
		x := 2
		inner = x
	}
	if inner == 2 && x == 1 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{13, `package main

func issue13Mutate(a [2]int) int { a[0] = 9; return a[0] + a[1] }

func main() {
	a := [2]int{1, 2}
	if issue13Mutate(a) == 11 && a[0] == 1 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{12, `package main

type issue12Pair struct{ a int; b string }

func main() {
	p := issue12Pair{a: 1, b: "x"}
	q := p
	q.a = 9
	if p.a == 1 && q.a == 9 && q.b == "x" { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{11, `package main

func main() {
	xs := make([]int, 1, 2)
	xs = append(xs, 8)
	xs = append(xs, 9)
	if len(xs) == 3 && cap(xs) >= 3 && xs[2] == 9 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{10, `package main

func main() {
	var s []int
	if len(s) == 0 && cap(s) == 0 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
		{8, `package main

func main() {
	s := []int{1}
	m := map[string]int{"x": s[0]}
	if m["x"] == 1 { print("PASS\n"); return }
	print("FAIL\n")
}
`},
	}

	var out []testCase
	for _, tc := range cases {
		out = append(out, simpleCase(legacyIssueTier(tc.issue), "legacy_regressions", tc.issue, tc.source))
	}
	out = append(out, moduleCase("quick", "legacy_regressions", 9, map[string]string{
		"cmd/app/config.go": `package main

var (
	base = 40
	zero int
)
`,
		"cmd/app/main.go": `package main

const inc = 2

func main() {
	base++
	if base+inc+zero == 43 { print("PASS\n"); return }
	print("FAIL\n")
}
`,
	}))
	return out
}

func legacyIssueTier(_ int) string {
	return "quick"
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

func quickLiterals(n int) []testCase {
	var out []testCase
	for i := 0; i < n; i++ {
		body := `package main

const hexValue = 0x3e

func append16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func main() {
	var out []byte
	out = append16(out, hexValue)
	if len(out) != 2 {
		print("FAIL\n")
		return
	}
	if out[0] != 0x3e {
		print("FAIL\n")
		return
	}
	if out[1] != 0 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`
		out = append(out, simpleCase("quick", "literals", i, body))
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
		if i == 0 {
			body = `package main

var trace int

func key(step int) string {
	trace = trace*10 + step
	if step == 1 { return "a" }
	if step == 2 { return "b" }
	return "c"
}

func main() {
	m := map[string]int{"a": 0, "b": 2, "c": 3}
	m[key(1)] = m[key(2)] + m[key(3)]
	if trace == 123 && m["a"] == 5 { print("PASS\n"); return }
	print("FAIL\n")
}
`
		}
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
		if i == 0 {
			body = `package main

type scorer interface { score() int }
type item struct { value int }

func mark(trace *int, step int) int { *trace = *trace*10+step; return 7 }
func (i item) score() int { return i.value }

func main() {
	trace := 0
	value := item{value: mark(&trace, 1)}
	var dynamic scorer = value
	got := dynamic.score()
	if trace == 1 && got == 7 { print("PASS\n"); return }
	print("FAIL\n")
}
`
		}
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
		if i == 0 {
			body = `package main

var trace int
func value(n int) int { trace = trace*10 + n; return n*10 }

func main() {
	a := [2]int{value(1), value(2)}
	if trace == 12 && a[0] == 10 && a[1] == 20 { print("PASS\n"); return }
	print("FAIL\n")
}
`
		}
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
		if i == 0 {
			body = `package main

func add(a int, b int) int { return a+b }
func mark(trace *int, step int) int { *trace = *trace*10+step; return step }
func apply(fn func(int, int) int, a int, b int) int { return fn(a, b) }

func main() {
	trace := 0
	fn := add
	left := mark(&trace, 2)
	right := mark(&trace, 3)
	got := apply(fn, left, right)
	if trace == 23 && got == 5 { print("PASS\n"); return }
	print("FAIL\n")
}
`
		}
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
		if i == 0 {
			body = `package main

func mark(trace *int, step int) int { *trace = *trace*10+step; return step }
func makeAdder(base int) func(int) int { return func(value int) int { return base+value } }

func main() {
	trace := 0
	next := makeAdder(mark(&trace, 1))
	got := next(mark(&trace, 2))
	if trace == 12 && got == 3 { print("PASS\n"); return }
	print("FAIL\n")
}
`
		}
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
	variant := "parameter_set"
	if index < 5 {
		variant = fmt.Sprintf("control_shape_%d", index)
	}
	if group == "legacy_regressions" {
		variant = "curated"
	}
	return testCase{tier: tier, group: group, name: name, files: files, variant: variant}
}

func writeCase(root string, tc testCase) {
	dir := filepath.Join(root, tc.tier, tc.group, tc.name)
	must(os.MkdirAll(dir, 0755))
	mod := tc.module
	if mod == "" {
		mod = modulePath(tc.tier, tc.group, caseIndex(tc.name))
	}
	must(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+mod+"\n\ngo 1.25\n"), 0644))
	for name, content := range tc.files {
		path := filepath.Join(dir, name)
		must(os.MkdirAll(filepath.Dir(path), 0755))
		if tc.group == "legacy_regressions" && strings.HasSuffix(name, ".go") {
			formatted, err := format.Source([]byte(content))
			must(err)
			content = string(formatted)
		} else if name == "cmd/app/main.go" && strings.HasPrefix(tc.variant, "control_shape_") {
			content = structuralVariant(content, caseIndex(tc.name)%5)
		}
		must(os.WriteFile(path, []byte(content), 0644))
	}
}

// structuralVariant rewrites only the final success/failure decision. The
// feature operation and its evaluation count stay unchanged while the corpus
// exercises distinct control-flow shapes around it.
func structuralVariant(source string, variant int) string {
	if variant == 0 {
		return source
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", source, parser.SkipObjectResolution)
	if err != nil {
		return source
	}
	var mainFn *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == "main" {
			mainFn = fn
			break
		}
	}
	if mainFn == nil || mainFn.Body == nil || len(mainFn.Body.List) < 2 {
		return source
	}
	last := len(mainFn.Body.List) - 1
	check, ok := mainFn.Body.List[last-1].(*ast.IfStmt)
	if !ok || check.Else != nil || len(check.Body.List) < 2 || !isTerminalReturn(check.Body.List[len(check.Body.List)-1]) || !isFailurePrint(mainFn.Body.List[last]) {
		return source
	}
	success := append([]ast.Stmt(nil), check.Body.List[:len(check.Body.List)-1]...)
	failure := mainFn.Body.List[last]
	if variant == 1 {
		mainFn.Body.List = append(mainFn.Body.List[:last-1],
			&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("corpusOK")}, Tok: token.DEFINE, Rhs: []ast.Expr{check.Cond}},
			&ast.IfStmt{Cond: &ast.UnaryExpr{Op: token.NOT, X: ast.NewIdent("corpusOK")}, Body: &ast.BlockStmt{List: []ast.Stmt{failure, &ast.ReturnStmt{}}}},
		)
		mainFn.Body.List = append(mainFn.Body.List, success...)
	} else if variant == 2 {
		mainFn.Body.List = append(mainFn.Body.List[:last-1], &ast.IfStmt{
			Cond: check.Cond,
			Body: &ast.BlockStmt{List: append(success, &ast.ReturnStmt{})},
			Else: &ast.BlockStmt{List: []ast.Stmt{failure}},
		})
	} else if variant == 3 {
		loopCheck := &ast.IfStmt{Cond: check.Cond, Body: &ast.BlockStmt{List: append(success, &ast.ReturnStmt{})}}
		loop := &ast.ForStmt{
			Init: &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("corpusAttempt")}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}}},
			Cond: &ast.BinaryExpr{X: ast.NewIdent("corpusAttempt"), Op: token.LSS, Y: &ast.BasicLit{Kind: token.INT, Value: "1"}},
			Post: &ast.IncDecStmt{X: ast.NewIdent("corpusAttempt"), Tok: token.INC},
			Body: &ast.BlockStmt{List: []ast.Stmt{loopCheck}},
		}
		mainFn.Body.List = append(mainFn.Body.List[:last-1], loop, failure)
	} else {
		mainFn.Body.List = append(mainFn.Body.List[:last-1],
			&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("corpusOK")}, Tok: token.DEFINE, Rhs: []ast.Expr{ast.NewIdent("false")}},
			&ast.IfStmt{Cond: check.Cond, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("corpusOK")}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent("true")}}}}},
			&ast.IfStmt{Cond: ast.NewIdent("corpusOK"), Body: &ast.BlockStmt{List: append(success, &ast.ReturnStmt{})}},
			failure,
		)
	}
	var out bytes.Buffer
	if format.Node(&out, fset, file) != nil {
		return source
	}
	return out.String()
}

func isTerminalReturn(stmt ast.Stmt) bool {
	ret, ok := stmt.(*ast.ReturnStmt)
	return ok && len(ret.Results) == 0
}

func isFailurePrint(stmt ast.Stmt) bool {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok || name.Name != "print" {
		return false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && strings.Contains(lit.Value, "FAIL")
}

func buildCorpusManifest(root string, cases []testCase) corpusManifest {
	type groupState struct {
		entry  corpusManifestGroup
		shapes map[string]bool
	}
	groups := make(map[string]*groupState)
	for _, tc := range cases {
		key := tc.tier + "/" + tc.group
		state := groups[key]
		if state == nil {
			state = &groupState{entry: corpusManifestGroup{Tier: tc.tier, Group: tc.group, Variants: make(map[string]int)}, shapes: make(map[string]bool)}
			groups[key] = state
		}
		state.entry.Cases++
		variant := tc.variant
		if variant == "" {
			variant = "curated"
		}
		state.entry.Variants[variant]++
		dir := filepath.Join(root, tc.tier, tc.group, tc.name)
		state.shapes[structuralFingerprint(dir)] = true
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	manifest := corpusManifest{Version: 1}
	for _, key := range keys {
		state := groups[key]
		state.entry.StructuralShapes = len(state.shapes)
		manifest.Groups = append(manifest.Groups, state.entry)
	}
	return manifest
}

func structuralFingerprint(dir string) string {
	var paths []string
	must(filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			paths = append(paths, path)
		}
		return nil
	}))
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		source, err := os.ReadFile(path)
		must(err)
		file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.SkipObjectResolution)
		must(err)
		hash.Write([]byte(filepath.Base(path)))
		ast.Inspect(file, func(node ast.Node) bool {
			if node != nil {
				fmt.Fprintf(hash, "%T;", node)
			}
			return true
		})
	}
	return hex.EncodeToString(hash.Sum(nil))
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
