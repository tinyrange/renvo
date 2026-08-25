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
	filter := flag.String("filter", "", "compile only target C sources containing this path fragment")
	objects := flag.Bool("objects", false, "emit an ELF object for every translation unit instead of syntax-checking it")
	installVmlinuxObjects := flag.Bool("install-vmlinux-objects", false, "replace the prepared tree's vmlinux C objects after a complete object pass")
	auditDirect := flag.Bool("audit-direct", false, "verify every recorded target C command used the selected driver and produced ELF")
	keepGoing := flag.Bool("keep-going", false, "continue after failures to enumerate the complete blocker set")
	flag.Parse()
	if *kernel == "" || *compiler == "" || *workers < 1 || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if *installVmlinuxObjects && !*objects {
		fatalf("-install-vmlinux-objects requires -objects")
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
	if *filter != "" {
		commands = filterTargetCommands(commands, *filter)
		if len(commands) == 0 {
			fatalf("no target C commands match filter %q", *filter)
		}
	}
	if *expected > 0 && len(commands) != *expected {
		fatalf("target C command count=%d, want %d", len(commands), *expected)
	}
	if *auditDirect {
		if err := auditDirectObjects(*kernel, compilerPath, commands); err != nil {
			fatalf("direct compiler audit: %v", err)
		}
		boot16 := m16TargetCommandCount(commands)
		fmt.Printf("gate=PASS\nmode=direct-audit\ncommands=%d\nrenvo_commands=%d\nrenvo_m16_commands=%d\n", len(commands), len(commands), boot16)
		return
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
		go syntaxWorker(ctx, &group, worker, *kernel, compilerPath, workspace, *objects, jobQueue, results)
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
	failureCount := 0
	var failed syntaxResult
	for result := range results {
		if result.err != nil {
			failureCount++
			if failed.err == nil {
				failed = result
			}
			if *keepGoing {
				fmt.Printf("failure=%d source=%s phase=%s error=%v\n%s", failureCount, result.job.source, result.phase, result.err, result.output)
			} else if failureCount == 1 {
				cancel()
			}
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
		if *keepGoing {
			_ = os.RemoveAll(workspace)
			fatalf("gate=FAIL\nchecked=%d/%d\nfailures=%d\nfirst=%s: %v", checked, len(commands), failureCount, failed.job.source, failed.err)
		}
		fatalf("%s failed %d/%d %s: %v\n%s", failed.phase, failed.job.index+1, len(commands), failed.job.source, failed.err, failed.output)
	}
	if *installVmlinuxObjects {
		installed, err := installVmlinuxTargetObjects(*kernel, workspace, jobs)
		if err != nil {
			fatalf("install target objects: %v", err)
		}
		fmt.Printf("installed=%d\n", installed)
	}
	mode := "syntax"
	if *objects {
		mode = "objects"
	}
	fmt.Printf("gate=PASS\nmode=%s\nelapsed=%s\n", mode, time.Since(started).Round(time.Millisecond))
}

func m16TargetCommandCount(commands []string) int {
	count := 0
	for _, command := range commands {
		for _, field := range strings.Fields(command) {
			if field == "-m16" {
				count++
				break
			}
		}
	}
	return count
}

func auditDirectObjects(kernel, compiler string, commands []string) error {
	for _, command := range commands {
		fields := strings.Fields(command)
		if len(fields) == 0 {
			return fmt.Errorf("empty saved command")
		}
		commandCompiler, err := filepath.Abs(fields[0])
		if err != nil {
			return fmt.Errorf("resolve command compiler %q: %w", fields[0], err)
		}
		if commandCompiler != compiler {
			return fmt.Errorf("command used %s, want %s: %s", commandCompiler, compiler, command)
		}
		object, ok := targetObjectPath(command)
		if !ok {
			return fmt.Errorf("unsupported object output: %s", command)
		}
		data, err := os.ReadFile(filepath.Join(kernel, object))
		if err != nil {
			return fmt.Errorf("read %s: %w", object, err)
		}
		if len(data) < 4 || string(data[:4]) != "\x7fELF" {
			return fmt.Errorf("%s is not ELF", object)
		}
	}
	return nil
}

func filterTargetCommands(commands []string, fragment string) []string {
	selected := make([]string, 0, len(commands))
	for _, command := range commands {
		source, _, ok := preprocessingCommand(command, "validate.i")
		if ok && strings.Contains(source, fragment) {
			selected = append(selected, command)
		}
	}
	return selected
}

func syntaxWorker(ctx context.Context, group *sync.WaitGroup, worker int, kernel string, compiler string, workspace string, objects bool, jobs <-chan syntaxJob, results chan<- syntaxResult) {
	defer group.Done()
	unit := filepath.Join(workspace, fmt.Sprintf("unit-%d.i", worker))
	for job := range jobs {
		if objects {
			// The stream is already preprocessed, but object-mode source loading
			// deliberately admits only ordinary C filenames. Giving the saved
			// stream a .c suffix exercises the exact production object path; with
			// no directives left, Renvo's preprocessing pass is an identity.
			unit = filepath.Join(workspace, fmt.Sprintf("unit-%03d.c", job.index))
		}
		_, preprocess, _ := preprocessingCommand(job.command, unit)
		cmd := exec.CommandContext(ctx, "sh", "-c", preprocess)
		cmd.Dir = kernel
		if output, err := cmd.CombinedOutput(); err != nil {
			results <- syntaxResult{job: job, phase: "system preprocessing", err: err, output: output}
			continue
		}
		arguments := []string{"cc", "-fsyntax-only", "-x", "c", unit}
		phase := "Renvo syntax check"
		object := ""
		if objects {
			object = filepath.Join(workspace, fmt.Sprintf("object-%03d.o", job.index))
			arguments = objectArguments(job.command, object, unit)
			phase = "Renvo object emission"
		}
		cmd = exec.CommandContext(ctx, compiler, arguments...)
		cmd.Dir = kernel
		if output, err := cmd.CombinedOutput(); err != nil {
			results <- syntaxResult{job: job, phase: phase, err: err, output: output}
			continue
		}
		if objects {
			data, err := os.ReadFile(object)
			if err != nil || len(data) < 4 || string(data[:4]) != "\x7fELF" {
				if err == nil {
					err = fmt.Errorf("output is not ELF")
				}
				results <- syntaxResult{job: job, phase: phase, err: err}
				continue
			}
		}
		results <- syntaxResult{job: job}
	}
}

func objectArguments(command, object, unit string) []string {
	arguments := []string{"cc", "-c", "-x", "c", "-o", object, unit}
	// Preserve the Kbuild flags that affect object semantics after the system
	// compiler has produced the saved preprocessing stream. Most warning and
	// optimization flags are intentionally absent at this boundary.
	if strings.Contains(command, " -fshort-wchar ") {
		arguments = append(arguments, "-fshort-wchar")
	}
	if strings.Contains(command, " -mcmodel=kernel ") {
		arguments = append(arguments, "-mcmodel=kernel")
	}
	if strings.Contains(command, " -m16 ") {
		arguments = append(arguments, "-m16")
	}
	return arguments
}

func targetObjectPath(command string) (string, bool) {
	compileAt := strings.LastIndex(command, " -c -o ")
	if compileAt < 0 {
		return "", false
	}
	fields := strings.Fields(command[compileAt+len(" -c -o "):])
	if len(fields) < 2 || filepath.IsAbs(fields[0]) {
		return "", false
	}
	path := filepath.Clean(fields[0])
	if path == "." || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", false
	}
	return path, true
}

func installVmlinuxTargetObjects(kernel, workspace string, jobs []syntaxJob) (int, error) {
	installed := 0
	for _, job := range jobs {
		// The x86 boot wrapper and vDSO are separate ELF environments with
		// their own compiler flags and linker scripts. This installation mode
		// replaces C objects consumed by vmlinux, while the corpus still
		// compiles and validates these independently linked sources.
		if strings.HasPrefix(job.source, "arch/x86/boot/") || strings.HasPrefix(job.source, "arch/x86/entry/vdso/") {
			continue
		}
		relative, ok := targetObjectPath(job.command)
		if !ok {
			return installed, fmt.Errorf("unsupported object output in command for %s", job.source)
		}
		source := filepath.Join(workspace, fmt.Sprintf("object-%03d.o", job.index))
		data, err := os.ReadFile(source)
		if err != nil {
			return installed, fmt.Errorf("read %s: %w", source, err)
		}
		destination := filepath.Join(kernel, relative)
		info, err := os.Lstat(destination)
		if err != nil {
			return installed, fmt.Errorf("inspect %s: %w", destination, err)
		}
		if !info.Mode().IsRegular() {
			return installed, fmt.Errorf("refuse to replace non-regular object %s", destination)
		}
		temporary, err := os.CreateTemp(filepath.Dir(destination), ".renvo-object-*")
		if err != nil {
			return installed, fmt.Errorf("stage %s: %w", destination, err)
		}
		temporaryName := temporary.Name()
		keep := false
		defer func() {
			if !keep {
				_ = os.Remove(temporaryName)
			}
		}()
		if err := temporary.Chmod(info.Mode().Perm()); err == nil {
			_, err = temporary.Write(data)
		}
		if closeErr := temporary.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return installed, fmt.Errorf("stage %s: %w", destination, err)
		}
		if err := os.Rename(temporaryName, destination); err != nil {
			return installed, fmt.Errorf("replace %s: %w", destination, err)
		}
		keep = true
		installed++
	}
	return installed, nil
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
			if at < 0 || !strings.HasPrefix(strings.TrimSpace(line[:at]), "savedcmd_") {
				continue
			}
			command := targetCompileCommand(strings.TrimSpace(line[at+2:]))
			fields := strings.Fields(command)
			if len(fields) > 1 && strings.Contains(command, " -nostdinc ") && strings.Contains(command, " -D__KERNEL__ ") &&
				strings.Contains(command, " -c ") && strings.HasSuffix(fields[len(fields)-1], ".c") {
				commands = append(commands, command)
			}
		}
		return scanner.Err()
	})
	sort.Strings(commands)
	return commands, err
}

func targetCompileCommand(command string) string {
	if at := strings.Index(command, " ; "); at >= 0 {
		command = command[:at]
	}
	return strings.TrimSpace(command)
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
	prefix := strings.Fields(command[:compileAt])
	filtered := prefix[:0]
	for i := 0; i < len(prefix); i++ {
		field := prefix[i]
		if strings.HasPrefix(field, "-Wp,-MMD,") || strings.HasPrefix(field, "-Wp,-MD,") || field == "-MMD" || field == "-MD" {
			continue
		}
		if field == "-MF" {
			i++
			continue
		}
		filtered = append(filtered, field)
	}
	return source, strings.Join(filtered, " ") + " -E -P " + source + " -o " + shellQuote(output), true
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func fatalf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
