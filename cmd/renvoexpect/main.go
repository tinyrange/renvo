package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const expectedOutput = "PASS\n"

type expectation struct {
	path    string
	backend bool
}

type backendExpectation struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func main() {
	root := flag.String("root", ".", "repository root")
	write := flag.Bool("write", false, "write the canonical positive-test expectations")
	flag.Parse()

	count, err := syncExpectations(*root, *write)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	action := "checked"
	if *write {
		action = "synchronized"
	}
	fmt.Printf("%s %d test expectations\n", action, count)
}

func syncExpectations(root string, write bool) (int, error) {
	expectations, err := discoverExpectations(root)
	if err != nil {
		return 0, err
	}
	for _, item := range expectations {
		if write {
			if _, err := os.Stat(item.path); os.IsNotExist(err) {
				if err := os.WriteFile(item.path, []byte(expectedOutput), 0o644); err != nil {
					return 0, fmt.Errorf("write %s: %w", item.path, err)
				}
			} else if err != nil {
				return 0, fmt.Errorf("stat %s: %w", item.path, err)
			}
		}
		data, err := os.ReadFile(item.path)
		if err != nil {
			return 0, fmt.Errorf("read %s: %w; run go run ./cmd/renvoexpect -write", item.path, err)
		}
		if err := validateExpectation(item, data); err != nil {
			return 0, err
		}
	}
	return len(expectations), nil
}

func discoverExpectations(root string) ([]expectation, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var found []expectation
	backendRoot := filepath.Join(root, "backend", "tests")
	err = filepath.WalkDir(backendRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			found = append(found, expectation{path: strings.TrimSuffix(path, ".go") + ".expected", backend: true})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover backend tests: %w", err)
	}

	for _, tier := range []string{"quick", "extended", "regressions", "std_compat"} {
		corpusRoot := filepath.Join(root, "frontend_tests", tier)
		err = filepath.WalkDir(corpusRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || entry.Name() != "go.mod" {
				return nil
			}
			dir := filepath.Dir(path)
			if info, statErr := os.Stat(filepath.Join(dir, "cmd", "app")); statErr != nil || !info.IsDir() {
				return nil
			}
			found = append(found, expectation{path: filepath.Join(dir, "expected.txt")})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("discover frontend %s tests: %w", tier, err)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].path < found[j].path })
	return found, nil
}

func validateExpectation(item expectation, data []byte) error {
	if !item.backend || len(data) == 0 || data[0] != '{' {
		if string(data) != expectedOutput {
			return fmt.Errorf("%s contains %q, want %q", item.path, string(data), expectedOutput)
		}
		return nil
	}
	var expected backendExpectation
	if err := json.Unmarshal(data, &expected); err != nil {
		return fmt.Errorf("decode %s: %w", item.path, err)
	}
	if expected.ExitCode != 0 || expected.Stdout+expected.Stderr != expectedOutput {
		return fmt.Errorf("%s contains invalid positive-test result: %#v", item.path, expected)
	}
	return nil
}
