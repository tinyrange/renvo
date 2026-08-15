// Command syntax checks every target C translation unit recorded by a prepared
// Kbuild tree. The original compiler command performs preprocessing; Renvo then
// parses and type-checks the resulting GNU C11 translation unit.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type syntaxJob struct {
	index   int
	source  string
	command string
}

type syntaxResult struct {
	job    syntaxJob
	phase  string
	err    error
	output []byte
}

func main() {
	kernel := flag.String("kernel", "", "prepared Linux build tree")
	compiler := flag.String("compiler", "", "Renvo compiler executable")
	expected := flag.Int("expected", 0, "required number of target C commands")
	workers := flag.Int("j", runtime.NumCPU(), "parallel preprocessing and syntax-check workers")
	flag.Parse()
	if *kernel == "" || *compiler == "" || *workers < 1 || flag.NArg() != 0 {
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
	jobs := make([]syntaxJob, len(commands))
	for i, command := range commands {
		source, _, ok := preprocessingCommand(command, filepath.Join(workspace, "validate.i"))
		if !ok {
			fatalf("target command has an unsupported compile suffix: %s", command)
		}
		jobs[i] = syntaxJob{index: i, source: source, command: command}
	}
	if *workers > len(jobs) {
		*workers = len(jobs)
	}
	started := time.Now()
	fmt.Printf("workspace=%s\ncommands=%d\nworkers=%d\n", workspace, len(commands), *workers)
	ctx, cancel := context.WithCancel(context.Background())
	jobQueue := make(chan syntaxJob)
	results := make(chan syntaxResult)
	var group sync.WaitGroup
	for worker := 0; worker < *workers; worker++ {
		group.Add(1)
		go syntaxWorker(ctx, &group, worker, *kernel, compilerPath, workspace, jobQueue, results)
	}
	go func() {
		defer close(jobQueue)
		for _, job := range jobs {
			select {
			case jobQueue <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		group.Wait()
		close(results)
	}()
	checked := 0
	var failed syntaxResult
	for result := range results {
		if result.err != nil && failed.err == nil {
			failed = result
			cancel()
			continue
		}
		if result.err == nil {
			checked++
			if checked%25 == 0 || checked == len(commands) {
				fmt.Printf("checked=%d/%d source=%s\n", checked, len(commands), result.job.source)
			}
		}
	}
	cancel()
	if failed.err != nil {
		fatalf("%s failed %d/%d %s: %v\n%s", failed.phase, failed.job.index+1, len(commands), failed.job.source, failed.err, failed.output)
	}
	fmt.Printf("gate=PASS\nelapsed=%s\n", time.Since(started).Round(time.Millisecond))
}

func syntaxWorker(ctx context.Context, group *sync.WaitGroup, worker int, kernel string, compiler string, workspace string, jobs <-chan syntaxJob, results chan<- syntaxResult) {
	defer group.Done()
	unit := filepath.Join(workspace, fmt.Sprintf("unit-%d.i", worker))
	for job := range jobs {
		_, preprocess, _ := preprocessingCommand(job.command, unit)
		cmd := exec.CommandContext(ctx, "sh", "-c", preprocess)
		cmd.Dir = kernel
		if output, err := cmd.CombinedOutput(); err != nil {
			results <- syntaxResult{job: job, phase: "system preprocessing", err: err, output: output}
			continue
		}
		cmd = exec.CommandContext(ctx, compiler, "cc", "-fsyntax-only", "-x", "c", unit)
		cmd.Dir = kernel
		if output, err := cmd.CombinedOutput(); err != nil {
			results <- syntaxResult{job: job, phase: "Renvo syntax check", err: err, output: output}
			continue
		}
		results <- syntaxResult{job: job}
	}
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
