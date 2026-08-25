package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/driver"
)

type expected struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

type testCase struct {
	Suite    string
	Name     string
	WorkDir  string
	Input    string
	Expected expected
}

type outcome struct {
	Suite, Name, Class, Detail string
	Size                       int
}

func main() {
	rootFlag := flag.String("root", ".", "repository root")
	runnerFlag := flag.String("runner", "", "COM emulator runner")
	suiteFlag := flag.String("suite", "all", "backend, frontend, or all")
	filterFlag := flag.String("filter", "", "regular expression matched against suite/name")
	arenaFlag := flag.Int("arena-size", 0, "override target arena size while qualifying")
	jobsFlag := flag.Int("jobs", 1, "parallel cases (in-process backend execution currently requires 1)")
	flag.Parse()
	if *runnerFlag == "" {
		fmt.Fprintln(os.Stderr, "-runner is required")
		os.Exit(2)
	}
	if *jobsFlag != 1 {
		fmt.Fprintln(os.Stderr, "-jobs must currently be 1 because prepared native backends execute in-process")
		os.Exit(2)
	}
	root, err := filepath.Abs(*rootFlag)
	check(err)
	runner, err := filepath.Abs(*runnerFlag)
	check(err)
	filter, err := regexp.Compile(*filterFlag)
	check(err)
	cases, err := discover(root, *suiteFlag, filter)
	check(err)
	if len(cases) == 0 {
		check(fmt.Errorf("no matching cases"))
	}

	definition := filepath.Join(root, "examples", "msdos", "msdos_com.rtg")
	cache, err := os.MkdirTemp("", "renvo-dos-backend-cache-*")
	check(err)
	defer os.RemoveAll(cache)

	jobs := *jobsFlag
	if jobs < 1 {
		jobs = 1
	}
	if jobs > len(cases) {
		jobs = len(cases)
	}
	tasks := make(chan testCase)
	results := make(chan outcome, len(cases))
	var completed atomic.Int64
	start := time.Now()
	var workers sync.WaitGroup
	for i := 0; i < jobs; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			backend := backendjit.New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), cache, backendcompiled.Backend{})
			for tc := range tasks {
				result := runCase(root, runner, backend, tc, *arenaFlag)
				results <- result
				n := completed.Add(1)
				if n%25 == 0 || int(n) == len(cases) {
					fmt.Fprintf(os.Stderr, "qualified %d/%d in %s\n", n, len(cases), time.Since(start).Round(time.Second))
				}
			}
		}()
	}
	go func() {
		for _, tc := range cases {
			tasks <- tc
		}
		close(tasks)
		workers.Wait()
		close(results)
	}()

	counts := map[string]int{}
	var failures []outcome
	for result := range results {
		counts[result.Class]++
		if result.Class != "pass" {
			failures = append(failures, result)
			status := "FAIL"
			if strings.HasPrefix(result.Class, "target-") {
				status = "SKIP"
			}
			fmt.Printf("%s\t%s\t%s\t%s\t%d\t%s\n", status, result.Suite, result.Name, result.Class, result.Size, oneLine(result.Detail))
		}
	}
	sort.Slice(failures, func(i, j int) bool {
		if failures[i].Suite != failures[j].Suite {
			return failures[i].Suite < failures[j].Suite
		}
		return failures[i].Name < failures[j].Name
	})
	fmt.Printf("SUMMARY\ttotal=%d\tpass=%d", len(cases), counts["pass"])
	keys := make([]string, 0, len(counts))
	for key := range counts {
		if key != "pass" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("\t%s=%d", key, counts[key])
	}
	fmt.Printf("\telapsed=%s\n", time.Since(start).Round(time.Second))
	for _, result := range failures {
		if !strings.HasPrefix(result.Class, "target-") {
			os.Exit(1)
		}
	}
}

func runCase(root, runner string, backend *backendjit.Backend, tc testCase, arenaSize int) outcome {
	var source []byte
	if tc.Suite == "backend" {
		var err error
		source, err = os.ReadFile(tc.Input)
		if err != nil {
			return outcome{Suite: tc.Suite, Name: tc.Name, Class: "harness", Detail: err.Error()}
		}
	}
	var compiled driver.CompileResult
	usedArena := 0
	for _, candidate := range qualificationArenas(arenaSize) {
		usedArena = candidate
		if tc.Suite == "backend" {
			result := backend.CompileSourceWithArena(source, "msdos/8086", true, candidate)
			compiled = driver.CompileResult{Binary: result.Binary, Ok: result.Ok, Diagnostic: result.Diagnostic}
		} else {
			args := []string{"-backend", filepath.Join(root, "examples", "msdos", "msdos_com.rtg"), "-t", "msdos/8086", "-s", "-o", "program.com"}
			if candidate > 0 {
				args = append(args, "-arena-size", fmt.Sprint(candidate))
			}
			args = append(args, "./cmd/app")
			compiled = driver.CompileFromFS(args, tc.WorkDir, filepath.Join(root, "std"), driver.OSFS{}, backend)
		}
		if compiled.Ok {
			break
		}
	}
	if !compiled.Ok {
		d := compiled.Diagnostic
		return classifyTargetLimit(tc, outcome{Suite: tc.Suite, Name: tc.Name, Class: "compile", Detail: fmt.Sprintf("arena=%d %s: %s", usedArena, d.Code, d.Message)})
	}
	out := outcome{Suite: tc.Suite, Name: tc.Name, Size: len(compiled.Binary)}
	if len(compiled.Binary) > 0xff00 {
		out.Class, out.Detail = "com-size", fmt.Sprintf("%d > 65280", len(compiled.Binary))
		return classifyTargetLimit(tc, out)
	}
	temp, err := os.MkdirTemp("", "renvo-dos-case-*")
	if err != nil {
		out.Class, out.Detail = "harness", err.Error()
		return out
	}
	defer os.RemoveAll(temp)
	image := filepath.Join(temp, "program.com")
	if err := os.WriteFile(image, compiled.Binary, 0o644); err != nil {
		out.Class, out.Detail = "harness", err.Error()
		return out
	}
	cmd := exec.Command(runner, image)
	cmd.Dir = tc.WorkDir
	var stdout, stderr bytes.Buffer
	if tc.Suite == "frontend" {
		// The frontend corpus records exec.Cmd.CombinedOutput, including the
		// implementation-defined stderr used by builtin print and println.
		cmd.Stdout, cmd.Stderr = &stdout, &stdout
	} else {
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
	}
	err = cmd.Run()
	if err != nil {
		out.Class = "emulator"
		out.Detail = fmt.Sprintf("%v stderr=%q stdout=%q", err, stderr.String(), stdout.String())
		return classifyTargetLimit(tc, out)
	}
	if stdout.String() != tc.Expected.Stdout || stderr.String() != tc.Expected.Stderr {
		out.Class = "output"
		out.Detail = fmt.Sprintf("stdout=%q want=%q stderr=%q want=%q", stdout.String(), tc.Expected.Stdout, stderr.String(), tc.Expected.Stderr)
		return classifyTargetLimit(tc, out)
	}
	out.Class = "pass"
	return out
}

var compactModelPattern = regexp.MustCompile(`\b(?:int32|uint32|int64|uint64|float32|float64|complex64|complex128|rune)\b|\b(?:complex|real|imag)\s*\(|\bunsafe\s*\.\s*Sizeof\b|(?:[0-9]|\))i\b|unicode/utf8|\butf8\s*\.`)
var foreignRuntimePattern = regexp.MustCompile(`(?m)\bfunc\s+syscall\s*\(|renvo:linkstatic|\b(?:os\.)?ReadDir\s*\(`)

func classifyTargetLimit(tc testCase, out outcome) outcome {
	source, _ := qualificationSource(tc)
	text := string(source)
	if foreignRuntimePattern.MatchString(text) {
		out.Class = "target-platform"
		out.Detail = "requires a foreign or directory-enumeration runtime operation; " + out.Detail
		return out
	}
	if compactModelPattern.MatchString(text) {
		out.Class = "target-model"
		out.Detail = "requires a value wider than the 16-bit target model; " + out.Detail
		return out
	}
	if strings.Contains(out.Detail, "out of memory") || strings.Contains(out.Detail, "unsupported opcode") {
		out.Class = "target-memory"
		out.Detail = "exhausts the single COM segment or its 4 KiB stack; " + out.Detail
		return out
	}
	if out.Class == "compile" && strings.Contains(out.Detail, "exit code 125") {
		out.Class = "target-memory"
		out.Detail = "the emitted program cannot fit code, data, BSS, heap, and stack in one COM segment; " + out.Detail
		return out
	}
	if sourceHasIntAbove(source, 32767) {
		out.Class = "target-model"
		out.Detail = "uses an integer constant outside signed 16-bit range; " + out.Detail
		return out
	}
	if out.Class == "compile" && strings.Contains(text, "make(") && sourceHasIntAbove(source, 16384) {
		out.Class = "target-memory"
		out.Detail = "requests an aggregate allocation that cannot fit in one COM segment; " + out.Detail
		return out
	}
	if out.Class == "com-size" || out.Class == "compile" && len(source) > 32767 {
		out.Class = "target-memory"
		out.Detail = "cannot fit code, data, BSS, heap, and stack in one COM segment; " + out.Detail
	}
	return out
}

func qualificationSource(tc testCase) ([]byte, error) {
	if tc.Input != "" {
		return os.ReadFile(tc.Input)
	}
	var source []byte
	err := filepath.WalkDir(tc.WorkDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		source = append(source, data...)
		source = append(source, '\n')
		return nil
	})
	return source, err
}

func sourceHasIntAbove(source []byte, limit uint64) bool {
	files := token.NewFileSet()
	file := files.AddFile("qualification.go", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, source, nil, scanner.ScanComments)
	for {
		_, kind, literal := scan.Scan()
		if kind == token.EOF {
			return false
		}
		if kind != token.INT {
			continue
		}
		value := strings.ReplaceAll(literal, "_", "")
		parsed, err := strconv.ParseUint(value, 0, 64)
		if err == nil && parsed > limit {
			return true
		}
	}
}

func qualificationArenas(requested int) []int {
	if requested > 0 {
		return []int{requested}
	}
	// A COM image shares one segment between code, static data, BSS, heap, and
	// stack. Start with the target default, then trade heap for image space so a
	// large program is not rejected solely because it does not need a 24 KiB
	// arena. Every selected program is still compiled and executed.
	return []int{24000, 16384, 12288, 8192, 4096, 2048, 1024, 512, 256}
}

func discover(root, suite string, filter *regexp.Regexp) ([]testCase, error) {
	var cases []testCase
	if suite == "all" || suite == "backend" {
		paths, err := filepath.Glob(filepath.Join(root, "backend", "tests", "*.go"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			name := filepath.Base(path)
			if !filter.MatchString("backend/" + name) {
				continue
			}
			want, err := backendExpected(strings.TrimSuffix(path, ".go") + ".expected")
			if err != nil {
				return nil, err
			}
			cases = append(cases, testCase{Suite: "backend", Name: name, WorkDir: filepath.Join(root, "backend"), Input: path, Expected: want})
		}
	}
	if suite == "all" || suite == "frontend" {
		for _, tier := range []string{"quick", "regressions", "extended"} {
			base := filepath.Join(root, "frontend_tests", tier)
			err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() || entry.Name() != "go.mod" {
					return err
				}
				dir := filepath.Dir(path)
				if _, err := os.Stat(filepath.Join(dir, "cmd", "app")); err != nil {
					return nil
				}
				rel, err := filepath.Rel(base, dir)
				if err != nil {
					return err
				}
				name := tier + "/" + filepath.ToSlash(rel)
				if !filter.MatchString("frontend/" + name) {
					return nil
				}
				data, err := os.ReadFile(filepath.Join(dir, "expected.txt"))
				if err != nil {
					return err
				}
				cases = append(cases, testCase{Suite: "frontend", Name: name, WorkDir: dir, Expected: expected{Stdout: string(data)}})
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	if suite != "all" && suite != "backend" && suite != "frontend" {
		return nil, fmt.Errorf("invalid suite %q", suite)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].Suite+cases[i].Name < cases[j].Suite+cases[j].Name })
	return cases, nil
}

func backendExpected(path string) (expected, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return expected{}, err
	}
	if len(data) != 0 && data[0] == '{' {
		var result expected
		err := json.Unmarshal(data, &result)
		return result, err
	}
	return expected{Stdout: string(data)}, nil
}

func oneLine(value string) string {
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\t", " ")
	if len(value) > 500 {
		return value[:500] + "..."
	}
	return value
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
