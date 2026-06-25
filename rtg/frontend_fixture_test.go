package rtg_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/build"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/rtgx"
	"j5.nz/rtg/rtg/unit"
)

const crossArchTestsEnv = "RTG_CROSS_ARCH_TESTS"

type frontendSmokeTarget struct {
	name   string
	runner []string
}

func TestHelloFixtureGoldenUnits(t *testing.T) {
	assertFixtureGoldenUnits(t, "hello_module")
}

func TestHelloFixtureFrontendMatchesHostGo(t *testing.T) {
	fixture := filepath.Join("testdata", "hello_module")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStructFixtureGoldenUnits(t *testing.T) {
	assertFixtureGoldenUnits(t, "struct_module")
}

func TestStructFixtureFrontendMatchesHostGo(t *testing.T) {
	fixture := filepath.Join("testdata", "struct_module")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nested\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/nested/pkg/dep"

func value() int {
	var total = dep.Join(dep.First(), dep.Second())
	if dep.Join(dep.First(), dep.Second()) == 12 {
		return total
	}
	return 0
}

func main() {
	if value() == 12 {
		dep.Emit(dep.First(), dep.Second())
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Second() int { return 2 }
func Join(a int, b int) int { return a*10 + b }
func Emit(a int, b int) int {
	if Join(a, b) == 12 {
		print("PASS\n")
	}
	return 0
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSwitchNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/switchcall\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/switchcall/pkg/dep"

func main() {
	switch dep.Join(dep.First(), dep.Second()) {
	case 12:
		dep.Emit()
	default:
		print("FAIL\n")
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Second() int { return 2 }
func Join(a int, b int) int { return a*10 + b }
func Emit() int {
	print("PASS\n")
	return 0
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestForConditionNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forcall\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forcall/pkg/dep"

func main() {
	count := 0
	for dep.KeepGoing(dep.Step()) {
		count = count + 1
		if count > 3 {
			print("FAIL\n")
			return
		}
	}
	if count == 2 && dep.Count() == 3 {
		dep.Emit()
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var count int
func Step() int {
	count = count + 1
	return count
}
func KeepGoing(v int) bool { return v < 3 }
func Count() int { return count }
func Emit() int {
	print("PASS\n")
	return 0
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForInitNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forinit\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forinit/pkg/dep"

func main() {
	total := 0
	for i := dep.Next(dep.First()); i < 4; i = i + 1 {
		total = total + i
	}
	if total == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Next(v int) int { return v + 1 }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForConditionNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forcond\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forcond/pkg/dep"

func main() {
	total := 0
	for i := 0; dep.KeepGoing(dep.Step()); i = i + 1 {
		total = total + i
		if i > 4 {
			print("FAIL\n")
			return
		}
	}
	if total == 3 && dep.Count() == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var count int
func Step() int {
	count = count + 1
	return count
}
func KeepGoing(v int) bool { return v < 4 }
func Count() int { return count }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForPostNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forpost\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forpost/pkg/dep"

func main() {
	total := 0
	for i := 0; i < 3; i = dep.Next(dep.Step()) {
		total = total + i
	}
	if total == 2 && dep.Count() == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var count int
func Step() int {
	count = count + 1
	return count
}
func Next(v int) int { return v + 1 }
func Count() int { return count }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSliceBoundCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/slicebounds\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/slicebounds/pkg/dep"

func main() {
	text := "xPASS\nx"
	print(text[dep.Start():dep.End()])
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func Start() int { return 1 }
func End() int { return 6 }
`)
	runFrontendFixtureOutput(t, fixture, []byte("PASS\n"))
}

func TestIfShortConditionNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/ifshortcond\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/ifshortcond/pkg/dep"

func main() {
	if total := dep.First(); dep.Check(dep.Next(total)) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Next(v int) int { return v + 1 }
func Check(v int) bool { return v == 2 }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSwitchShortTagNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/switchshorttag\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/switchshorttag/pkg/dep"

func main() {
	switch total := dep.First(); dep.Next(total) {
	case 2:
		print("PASS\n")
		return
	default:
		print("FAIL\n")
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Next(v int) int { return v + 1 }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdFmtPrintlnFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdprint\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "fmt"

func main() {
	fmt.Print("PA")
	fmt.Print("SS")
	fmt.Print("\n")
	fmt.Println("PASS")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdOSFileIntrinsicsFrontendRuns(t *testing.T) {
	fixture := t.TempDir()
	dataPath := filepath.ToSlash(filepath.Join(fixture, "rtg_std_os.tmp"))
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdos\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "os"

func fail(msg string) {
	os.Write(os.Stdout, []byte(msg), -1)
}

func main() {
	fd := os.Open(`+strconv.Quote(dataPath)+`, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if fd < 0 {
		fail("FAIL open\n")
		return
	}
	var data []byte
	data = append(data, 'O')
	data = append(data, 'K')
	if os.Write(fd, data, int64(0)) != 2 {
		fail("FAIL write\n")
		os.Close(fd)
		return
	}
	var readBuf []byte
	readBuf = append(readBuf, 0)
	readBuf = append(readBuf, 0)
	if os.Read(fd, readBuf, int64(0)) != 2 {
		fail("FAIL read\n")
		os.Close(fd)
		return
	}
	if os.Chmod(fd, 420) != 0 {
		fail("FAIL chmod\n")
		os.Close(fd)
		return
	}
	os.Close(fd)
	if readBuf[0] == 'O' && readBuf[1] == 'K' {
		os.Write(os.Stdout, []byte("PASS\n"), -1)
		return
	}
	fail("FAIL data\n")
}
`)
	runFrontendFixtureOutput(t, fixture, []byte("PASS\n"))
}

func TestStdStringsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdstrings\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "strings"

func main() {
	value := "prefix-body-suffix"
	ok := strings.HasPrefix(value, "prefix")
	ok = ok && strings.HasSuffix(value, "suffix")
	ok = ok && strings.Contains(value, "body")
	ok = ok && !strings.Contains(value, "missing")
	ok = ok && strings.Contains(value, "")
	ok = ok && strings.IndexByte(value, '-') == 6
	ok = ok && strings.IndexByte(value, 'z') == -1
	ok = ok && strings.Index(value, "body") == 7
	ok = ok && strings.Index(value, "missing") == -1
	ok = ok && strings.TrimPrefix(value, "prefix-") == "body-suffix"
	ok = ok && strings.TrimSuffix(value, "-suffix") == "prefix-body"
	ok = ok && strings.TrimSpace(" \tPASS\n") == "PASS"
	parts := strings.Split("red,green,blue", ",")
	ok = ok && len(parts) == 3 && parts[0] == "red" && parts[2] == "blue"
	fields := strings.Fields("  alpha\tbeta\n gamma ")
	ok = ok && len(fields) == 3 && fields[1] == "beta"
	ok = ok && strings.Join(fields, "-") == "alpha-beta-gamma"
	if ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdStrconvFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdstrconv\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "strconv"

func main() {
	ok := strconv.Itoa(0) == "0"
	ok = ok && strconv.Itoa(12345) == "12345"
	ok = ok && strconv.Itoa(-42) == "-42"
	ok = ok && strconv.Quote("PASS\n") == "\"PASS\\n\""
	ok = ok && strconv.Quote("a\\b") == "\"a\\\\b\""
	if ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdPathFilepathFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdfilepath\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "path/filepath"

func main() {
	ok := filepath.Separator == '/'
	ok = ok && filepath.ToSlash("a/b") == "a/b"
	ok = ok && filepath.FromSlash("a/b") == "a/b"
	ok = ok && filepath.IsAbs("/tmp")
	ok = ok && !filepath.IsAbs("tmp")
	ok = ok && filepath.Clean("/a//b/./c/..") == "/a/b"
	ok = ok && filepath.Clean("a/../b") == "b"
	ok = ok && filepath.Join("/tmp/", "app") == "/tmp/app"
	ok = ok && filepath.Base("/tmp/app/main.go") == "main.go"
	ok = ok && filepath.Dir("/tmp/app/main.go") == "/tmp/app"
	ok = ok && filepath.Ext("/tmp/app/main.go") == ".go"
	if ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdBytesBufferFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdbytes\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "bytes"

func main() {
	var b bytes.Buffer
	b.WriteString("PA")
	b.WriteByte('S')
	b.WriteByte('S')
	if b.Len() != 4 {
		print("FAIL len\n")
		return
	}
	if string(b.Bytes()) != "PASS" {
		print("FAIL bytes\n")
		return
	}
	b.WriteString("\n")
	if b.String() == "PASS\n" {
		print(b.String())
		return
	}
	print("FAIL string\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdRuntimeFrontendMatchesHostGo(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("native runtime fixture expects linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdruntime\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "runtime"

func main() {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdRuntimeFrontendTargetConstants(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdruntimetarget\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "runtime"

func main() {
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	units := loadFixtureUnitsForTarget(t, fixture, "linux/aarch64")
	if !unitsContainBody(units, `const rtg_runtime_GOOS = "linux"`) {
		t.Fatalf("linux/aarch64 units did not include linux GOOS: %#v", units)
	}
	if !unitsContainBody(units, `const rtg_runtime_GOARCH = "arm64"`) {
		t.Fatalf("linux/aarch64 units did not include arm64 GOARCH: %#v", units)
	}
	if unitsContainBody(units, `const rtg_runtime_GOARCH = "amd64"`) {
		t.Fatalf("linux/aarch64 units included amd64 GOARCH: %#v", units)
	}
}

func TestMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/methods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type point struct {
	x int
	y int
}

func (p point) Sum() int {
	return p.x + p.y
}

func main() {
	p := point{x: 3, y: 4}
	if p.Sum() == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPointerReceiverMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/ptrmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type counter struct {
	n int
}

func (c *counter) Add(v int) {
	c.n = c.n + v
}

func main() {
	var c counter
	c.Add(7)
	c.Add(5)
	if c.n == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestTopLevelNameListsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namelists\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/namelists/pkg/dep"

func main() {
	dep.C = 3
	dep.D = 4
	if dep.Sum() == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

const A, B = 1, 2
var C, D int

func Sum() int {
	return A + B + C + D
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestFrontendUnitsUseRequestedTargetFiles(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/targetfiles\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/targetfiles/pkg/dep"

func main() {
	if dep.Value() == 1 {
		print("PASS\n")
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/value_amd64.go", `package dep

func Value() int { return 1 }
`)
	writeFixtureFile(t, fixture, "pkg/dep/value_arm64.go", `package dep

func Value() int { return 2 }
`)
	amd64Units := loadFixtureUnitsForTarget(t, fixture, "linux/amd64")
	aarch64Units := loadFixtureUnitsForTarget(t, fixture, "linux/aarch64")
	if !unitsContainBody(amd64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 1 }") {
		t.Fatalf("linux/amd64 units did not include amd64 body: %#v", amd64Units)
	}
	if unitsContainBody(amd64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 2 }") {
		t.Fatalf("linux/amd64 units included arm64 body: %#v", amd64Units)
	}
	if !unitsContainBody(aarch64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 2 }") {
		t.Fatalf("linux/aarch64 units did not include arm64 body: %#v", aarch64Units)
	}
	if unitsContainBody(aarch64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 1 }") {
		t.Fatalf("linux/aarch64 units included amd64 body: %#v", aarch64Units)
	}
}

func runFrontendFixtureMatchesHostGo(t *testing.T, fixture string) {
	t.Helper()
	host := exec.Command("go", "run", "./cmd/app")
	host.Dir = fixture
	hostOut, err := host.CombinedOutput()
	if err != nil {
		t.Fatalf("host fixture failed: %v\n%s", err, string(hostOut))
	}
	for _, target := range frontendSmokeTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			if len(target.runner) > 0 {
				if _, err := exec.LookPath(target.runner[0]); err != nil {
					t.Skipf("runner %s is not installed", target.runner[0])
				}
			}
			units := loadFixtureUnitsForTarget(t, fixture, target.name)
			out := filepath.Join(t.TempDir(), "hello")
			if err := rtgx.CompileUnits(units, rtgx.Options{Target: target.name, Output: out}); err != nil {
				t.Fatalf("CompileUnits failed: %v", err)
			}
			frontOut, err := runFrontendTarget(target, out)
			if err != nil {
				t.Fatalf("frontend fixture failed: %v\n%s", err, string(frontOut))
			}
			if !bytes.Equal(frontOut, hostOut) {
				t.Fatalf("frontend output = %q, host output = %q", string(frontOut), string(hostOut))
			}
		})
	}
}

func runFrontendFixtureOutput(t *testing.T, fixture string, want []byte) {
	t.Helper()
	for _, target := range frontendSmokeTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			if len(target.runner) > 0 {
				if _, err := exec.LookPath(target.runner[0]); err != nil {
					t.Skipf("runner %s is not installed", target.runner[0])
				}
			}
			units := loadFixtureUnitsForTarget(t, fixture, target.name)
			out := filepath.Join(t.TempDir(), "app")
			if err := rtgx.CompileUnits(units, rtgx.Options{Target: target.name, Output: out}); err != nil {
				t.Fatalf("CompileUnits failed: %v", err)
			}
			frontOut, err := runFrontendTarget(target, out)
			if err != nil {
				t.Fatalf("frontend fixture failed: %v\n%s", err, string(frontOut))
			}
			if !bytes.Equal(frontOut, want) {
				t.Fatalf("frontend output = %q, want %q", string(frontOut), string(want))
			}
		})
	}
}

func unitsContainBody(units []unit.Unit, body string) bool {
	for _, u := range units {
		for _, decl := range u.Decls {
			if strings.Contains(decl.Body, body) {
				return true
			}
		}
	}
	return false
}

func assertFixtureGoldenUnits(t *testing.T, name string) {
	t.Helper()
	fixture := filepath.Join("testdata", name)
	units := loadFixtureUnits(t, fixture)
	for _, u := range units {
		unitName := emit.FileName(u.ImportPath)
		got := string(emit.Source(u))
		wantPath := filepath.Join("testdata", "golden", name, unitName)
		want, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("ReadFile golden %s failed: %v", wantPath, err)
		}
		if got != string(want) {
			t.Fatalf("golden mismatch for %s\n%s", unitName, diffText(string(want), got))
		}
	}
}

func writeFixtureFile(t *testing.T, root string, path string, data string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll %s failed: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", full, err)
	}
}

func frontendSmokeTargets(t *testing.T) []frontendSmokeTarget {
	t.Helper()
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		targets := []frontendSmokeTarget{{name: "linux/amd64"}}
		if os.Getenv(crossArchTestsEnv) == "1" {
			targets = append(targets,
				frontendSmokeTarget{name: "linux/386"},
				frontendSmokeTarget{name: "linux/aarch64", runner: []string{"qemu-aarch64"}},
				frontendSmokeTarget{name: "linux/arm", runner: []string{"qemu-arm"}},
				frontendSmokeTarget{name: "wasi/wasm32", runner: []string{"wasmtime", "run", "--dir=.", "--dir=/", "--env", "PWD", "--env", "PATH"}},
			)
		}
		return targets
	case "linux/arm64":
		return []frontendSmokeTarget{{name: "linux/aarch64"}}
	default:
		t.Skipf("no frontend smoke targets supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
}

func runFrontendTarget(target frontendSmokeTarget, path string) ([]byte, error) {
	if len(target.runner) == 0 {
		cmd := exec.Command(path)
		return cmd.CombinedOutput()
	}
	args := append([]string{}, target.runner[1:]...)
	args = append(args, path)
	cmd := exec.Command(target.runner[0], args...)
	return cmd.CombinedOutput()
}

func loadFixtureUnits(t *testing.T, fixture string) []unit.Unit {
	t.Helper()
	return loadFixtureUnitsForTarget(t, fixture, "")
}

func loadFixtureUnitsForTarget(t *testing.T, fixture string, target string) []unit.Unit {
	t.Helper()
	graph, err := load.LoadEntries([]string{filepath.Join(fixture, "cmd", "app")}, load.Options{Target: target})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	units, err := build.Units(graph)
	if err != nil {
		t.Fatalf("build.Units failed: %v", err)
	}
	sort.Slice(units, func(i int, j int) bool {
		return units[i].ImportPath < units[j].ImportPath
	})
	return units
}

func diffText(want string, got string) string {
	if want == got {
		return ""
	}
	return "want:\n" + want + "\ngot:\n" + got
}
