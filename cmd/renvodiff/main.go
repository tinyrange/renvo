//go:build !renvo

// Command renvodiff searches for semantic differences between host Go and
// Renvo, and reduces each difference to a small standalone program.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"renvo.dev/internal/difftest"
)

func main() {
	flags := flag.NewFlagSet("renvodiff", flag.ExitOnError)
	seed := flags.Uint64("seed", 1, "first deterministic generator seed")
	count := flags.Int("count", 1000, "number of generated programs")
	cases := flags.Int("cases", 1, "independently removable feature cases per generated program")
	family := flags.String("family", "", "restrict generation to one feature family")
	listFamilies := flags.Bool("list-families", false, "list feature families and exit")
	timeout := flags.Duration("timeout", 3*time.Second, "execution timeout per compiler output")
	output := flags.String("out", "sandbox/difftest", "directory for discrepancies")
	stdRoot := flags.String("std", "std", "Renvo standard-library root")
	target := flags.String("target", difftest.HostTarget(), "runnable Renvo host target")
	minimizePath := flags.String("minimize", "", "reduce an existing discrepant Go source file")
	minimizeFindings := flags.Bool("minimize-findings", true, "minimize each newly discovered discrepancy")
	progress := flags.Int("progress", 25, "print progress after this many programs (zero disables)")
	stopAfter := flags.Int("stop-after", 0, "stop after this many discrepancies (zero continues)")
	flags.Parse(os.Args[1:])
	if *listFamilies {
		for _, name := range difftest.Families() {
			fmt.Println(name)
		}
		return
	}

	runner := difftest.Runner{StdRoot: absolute(*stdRoot), Target: *target, Timeout: *timeout}
	if *minimizePath != "" {
		if err := reduceFile(runner, *minimizePath); err != nil {
			fmt.Fprintln(os.Stderr, "renvodiff:", err)
			os.Exit(1)
		}
		return
	}
	if *target == "" {
		fmt.Fprintln(os.Stderr, "renvodiff: this host has no runnable Renvo target")
		os.Exit(2)
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "renvodiff:", err)
		os.Exit(1)
	}

	started := time.Now()
	found := 0
	for index := 0; index < *count; index++ {
		currentSeed := *seed + uint64(index)
		var source []byte
		var err error
		if *family == "" {
			source, err = difftest.Generate(currentSeed, *cases)
		} else {
			source, err = difftest.GenerateFamily(currentSeed, *cases, *family)
		}
		if err != nil {
			fatalSeed(currentSeed, err)
		}
		comparison, err := runner.Compare(source)
		if err != nil {
			fatalSeed(currentSeed, err)
		}
		if comparison.Interesting {
			found++
			if err := saveFinding(runner, *output, currentSeed, source, comparison, *minimizeFindings); err != nil {
				fatalSeed(currentSeed, err)
			}
			if *stopAfter > 0 && found >= *stopAfter {
				break
			}
		}
		if *progress > 0 && (index+1)%*progress == 0 {
			fmt.Printf("tested=%d findings=%d rate=%.1f programs/s\n", index+1, found, float64(index+1)/time.Since(started).Seconds())
		}
	}
	fmt.Printf("complete findings=%d elapsed=%s\n", found, time.Since(started).Round(time.Millisecond))
}

func saveFinding(runner difftest.Runner, root string, seed uint64, source []byte, comparison difftest.Comparison, minimize bool) error {
	dir := filepath.Join(root, fmt.Sprintf("seed-%016x", seed))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "original.go"), source, 0o644); err != nil {
		return err
	}
	signature := comparison.Signature
	minimized := source
	final := comparison
	if minimize {
		var err error
		minimized, err = difftest.Minimize(source, func(candidate []byte) (bool, error) {
			result, compareErr := runner.Compare(candidate)
			return compareErr == nil && result.Interesting && result.Signature == signature, compareErr
		})
		if err != nil {
			return err
		}
		final, err = runner.Compare(minimized)
		if err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "minimized.go"), minimized, 0o644); err != nil {
		return err
	}
	report := fmt.Sprintf("seed: %d\nsignature: %s\nhost exit: %d\nhost output: %q\nrenvo exit: %d\nrenvo output: %q\nrenvo diagnostic: %s\n", seed, final.Signature, final.Host.ExitCode, final.Host.Output, final.Renvo.ExitCode, final.Renvo.Output, final.Renvo.Diagnostic)
	if err := os.WriteFile(filepath.Join(dir, "report.txt"), []byte(report), 0o644); err != nil {
		return err
	}
	fmt.Printf("FOUND seed=%d signature=%s source=%s\n", seed, signature, filepath.Join(dir, "minimized.go"))
	return nil
}

func reduceFile(runner difftest.Runner, path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	initial, err := runner.Compare(source)
	if err != nil {
		return err
	}
	if !initial.Interesting {
		return fmt.Errorf("%s does not produce a discrepancy", path)
	}
	minimized, err := difftest.Minimize(source, func(candidate []byte) (bool, error) {
		comparison, compareErr := runner.Compare(candidate)
		return compareErr == nil && comparison.Interesting && comparison.Signature == initial.Signature, compareErr
	})
	if err != nil {
		return err
	}
	output := path + ".min.go"
	if err := os.WriteFile(output, minimized, 0o644); err != nil {
		return err
	}
	fmt.Println(output)
	return nil
}

func absolute(path string) string {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return resolved
}

func fatalSeed(seed uint64, err error) {
	fmt.Fprintln(os.Stderr, "renvodiff seed "+strconv.FormatUint(seed, 10)+":", err)
	os.Exit(1)
}
