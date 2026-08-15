// Command syntax checks every target C translation unit recorded by a prepared
// Kbuild tree. The original compiler command performs preprocessing; Renvo then
// parses and type-checks the resulting GNU C11 translation unit.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	kernel := flag.String("kernel", "", "prepared Linux build tree")
	compiler := flag.String("compiler", "", "Renvo compiler executable")
	expected := flag.Int("expected", 0, "required number of target C commands")
	flag.Parse()
	if *kernel == "" || *compiler == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	compilerPath, err := exec.LookPath(*compiler)
	if err != nil {
		fatalf("resolve compiler: %v", err)
	}
	compilerPath, err = filepath.Abs(compilerPath)
	if err != nil {
		fatalf("resolve compiler path: %v", err)
	}

	commands, err := targetCCommands(*kernel)
	if err != nil {
		fatalf("read Kbuild commands: %v", err)
	}
	if len(commands) == 0 {
		fatalf("no target C commands found under %s; build the pinned tinyconfig with the system compiler first", *kernel)
	}
	if *expected > 0 && len(commands) != *expected {
		fatalf("target C command count=%d, want %d", len(commands), *expected)
	}

	workspace, err := os.MkdirTemp("", "renvo-linux-syntax-")
	if err != nil {
		fatalf("create workspace: %v", err)
	}
	// A .i suffix tells the compiler that the system compiler has already
	// performed translation phases 1-4. This gate measures Renvo's M4 parser
	// and semantic checker rather than redundantly preprocessing the result.
	unit := filepath.Join(workspace, "unit.i")
	started := time.Now()
	fmt.Printf("workspace=%s\ncommands=%d\n", workspace, len(commands))
	for i, command := range commands {
		source, preprocess, ok := preprocessingCommand(command, unit)
		if !ok {
			fatalf("target command has an unsupported compile suffix: %s", command)
		}
		cmd := exec.Command("sh", "-c", preprocess)
		cmd.Dir = *kernel
		if output, err := cmd.CombinedOutput(); err != nil {
			fatalf("system preprocessing failed %d/%d %s: %v\n%s", i+1, len(commands), source, err, output)
		}
		cmd = exec.Command(compilerPath, "cc", "-fsyntax-only", "-x", "c", unit)
		cmd.Dir = *kernel
		if output, err := cmd.CombinedOutput(); err != nil {
			fatalf("Renvo syntax check failed %d/%d %s: %v\n%s", i+1, len(commands), source, err, output)
		}
		if (i+1)%25 == 0 || i+1 == len(commands) {
			fmt.Printf("checked=%d/%d source=%s\n", i+1, len(commands), source)
		}
	}
	fmt.Printf("gate=PASS\nelapsed=%s\n", time.Since(started).Round(time.Millisecond))
}

func targetCCommands(root string) ([]string, error) {
	var commands []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".cmd") {
			return walkErr
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			at := strings.Index(line, ":=")
			if at < 0 || !strings.Contains(line[:at], "cmd_") {
				continue
			}
			command := strings.TrimSpace(line[at+2:])
			fields := strings.Fields(command)
			if len(fields) > 1 && strings.Contains(command, " -nostdinc ") &&
				strings.Contains(command, " -c ") && strings.HasSuffix(fields[len(fields)-1], ".c") {
				commands = append(commands, command)
			}
		}
		return scanner.Err()
	})
	sort.Strings(commands)
	return commands, err
}

func preprocessingCommand(command, output string) (string, string, bool) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "", "", false
	}
	source := fields[len(fields)-1]
	compileAt := strings.LastIndex(command, " -c -o ")
	if compileAt < 0 {
		return source, "", false
	}
	return source, command[:compileAt] + " -E -P " + source + " -o " + shellQuote(output), true
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func fatalf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
