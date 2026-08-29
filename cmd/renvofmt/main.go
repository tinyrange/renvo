// Command renvofmt formats and validates RTG and RBE source files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgformat"
)

func main() {
	write := flag.Bool("w", false, "write result to each source file")
	list := flag.Bool("l", false, "list files whose formatting differs")
	check := flag.Bool("check", false, "fail if any source file is invalid or not formatted")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: renvofmt [-w] [-l] [-check] path ...")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if *write && *check {
		fail("-w and -check cannot be used together")
	}
	paths, err := sourcePaths(flag.Args())
	if err != nil {
		fail(err.Error())
	}
	if len(paths) == 0 {
		fail("no .rtg or .rbe files found")
	}
	if !*write && !*list && !*check && len(paths) != 1 {
		fail("standard-output mode accepts exactly one source file")
	}

	failed := false
	for _, path := range paths {
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			fmt.Fprintln(os.Stderr, readErr)
			failed = true
			continue
		}
		formatted, formatErr := rtgformat.Source(source, path, filesystemImportLoader{})
		if formatErr != nil {
			fmt.Fprintln(os.Stderr, formatErr)
			failed = true
			continue
		}
		changed := !bytes.Equal(source, formatted)
		if changed && (*list || *check) {
			fmt.Println(path)
		}
		if *write && changed {
			mode := os.FileMode(0o644)
			if info, statErr := os.Stat(path); statErr == nil {
				mode = info.Mode().Perm()
			}
			if writeErr := os.WriteFile(path, formatted, mode); writeErr != nil {
				fmt.Fprintln(os.Stderr, writeErr)
				failed = true
			}
		} else if !*write && !*list && !*check {
			if _, writeErr := os.Stdout.Write(formatted); writeErr != nil {
				fmt.Fprintln(os.Stderr, writeErr)
				failed = true
			}
		}
		if *check && changed {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func sourcePaths(arguments []string) ([]string, error) {
	seen := make(map[string]bool)
	var paths []string
	for _, argument := range arguments {
		info, err := os.Stat(argument)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if !rtgformat.Extension(argument) {
				return nil, fmt.Errorf("%s is not an .rtg or .rbe file", argument)
			}
			clean := filepath.Clean(argument)
			if !seen[clean] {
				seen[clean] = true
				paths = append(paths, clean)
			}
			continue
		}
		err = filepath.WalkDir(argument, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != argument && (entry.Name() == ".git" || entry.Name() == "sandbox") {
					return filepath.SkipDir
				}
				return nil
			}
			if rtgformat.Extension(path) {
				clean := filepath.Clean(path)
				if !seen[clean] {
					seen[clean] = true
					paths = append(paths, clean)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(paths)
	return paths, nil
}

type filesystemImportLoader struct{}

func (filesystemImportLoader) LoadImport(importingFilename, importPath string) rtg.ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, err := os.ReadFile(path)
	return rtg.ImportSource{Source: source, Filename: path, Ok: err == nil}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "renvofmt:", message)
	os.Exit(2)
}
