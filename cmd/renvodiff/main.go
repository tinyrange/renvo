//go:build !renvo

// Command renvodiff searches for semantic differences between host Go and
// Renvo, and reduces each difference to a small standalone program.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"renvo.dev/internal/difftest"
)

func main() {
	flags := flag.NewFlagSet("renvodiff", flag.ExitOnError)
	seed := flags.Uint64("seed", 1, "first deterministic generator seed")
	count := flags.Int("count", 1000, "number of generated programs")
	cases := flags.Int("cases", 1, "independently removable feature cases per generated program")
	family := flags.String("family", "", "restrict generation to one feature family")
	swarm := flags.Bool("swarm", false, "use a seed-specific subset of feature families")
	policy := flags.String("policy", "", "use a named code-shape generation policy")
	listFamilies := flags.Bool("list-families", false, "list feature families and exit")
	listPolicies := flags.Bool("list-policies", false, "list generation policies and exit")
	timeout := flags.Duration("timeout", 3*time.Second, "execution timeout per compiler output")
	output := flags.String("out", "sandbox/difftest", "directory for discrepancies")
	stdRoot := flags.String("std", "std", "Renvo standard-library root")
	target := flags.String("target", difftest.HostTarget(), "runnable Renvo host target")
	targets := flags.String("targets", "", "comma-separated runnable Renvo target matrix")
	metamorphic := flags.Bool("metamorphic", false, "test semantics-preserving source variants")
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
	if *listPolicies {
		for _, name := range difftest.Policies() {
			fmt.Println(name)
		}
		return
	}

	targetNames := []string{*target}
	if *targets != "" {
		targetNames = targetNames[:0]
		for _, name := range strings.Split(*targets, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				targetNames = append(targetNames, name)
			}
		}
	}
	if len(targetNames) == 0 || targetNames[0] == "" {
		fmt.Fprintln(os.Stderr, "renvodiff: this host has no runnable Renvo target")
		os.Exit(2)
	}
	for _, name := range targetNames {
		if err := difftest.TargetRunnable(name); err != nil {
			fmt.Fprintln(os.Stderr, "renvodiff:", err)
			os.Exit(2)
		}
	}
	runner := difftest.Runner{StdRoot: absolute(*stdRoot), Target: targetNames[0], Timeout: *timeout}
	if *minimizePath != "" {
		if err := reduceFile(runner, *minimizePath); err != nil {
			fmt.Fprintln(os.Stderr, "renvodiff:", err)
			os.Exit(1)
		}
		return
	}
	if (*family != "" && *swarm) || (*policy != "" && *swarm) {
		fmt.Fprintln(os.Stderr, "renvodiff: -swarm cannot be combined with -family or -policy")
		os.Exit(2)
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "renvodiff:", err)
		os.Exit(1)
	}

	started := time.Now()
	found := 0
	stop := false
	for index := 0; index < *count; index++ {
		currentSeed := *seed + uint64(index)
		var source []byte
		var err error
		if *policy != "" && *family != "" {
			source, err = difftest.GenerateFamilyPolicy(currentSeed, *cases, *family, *policy)
		} else if *policy != "" {
			source, err = difftest.GeneratePolicy(currentSeed, *cases, *policy)
		} else if *swarm {
			source, err = difftest.GenerateSwarm(currentSeed, *cases)
		} else if *family == "" {
			source, err = difftest.Generate(currentSeed, *cases)
		} else {
			source, err = difftest.GenerateFamily(currentSeed, *cases, *family)
		}
		if err != nil {
			fatalSeed(currentSeed, err)
		}
		variants := []difftest.Variant{{Name: "original", Source: source}}
		if *metamorphic {
			transformed, variantErr := difftest.Variants(source, currentSeed)
			if variantErr != nil {
				fatalSeed(currentSeed, variantErr)
			}
			variants = append(variants, transformed...)
		}
		var baselineHost difftest.Execution
		baselineSet := false
		for _, variant := range variants {
			for _, targetName := range targetNames {
				activeRunner := runner
				activeRunner.Target = targetName
				comparison, compareErr := activeRunner.Compare(variant.Source)
				if compareErr != nil {
					fatalSeed(currentSeed, compareErr)
				}
				if !baselineSet {
					baselineHost = comparison.Host
					baselineSet = true
				} else if variant.Name != "original" && !equivalentExecution(baselineHost, comparison.Host) {
					fatalSeed(currentSeed, fmt.Errorf("metamorphic variant %s changed host behavior: baseline=%q variant=%q", variant.Name, baselineHost.Output, comparison.Host.Output))
				}
				if comparison.Interesting {
					found++
					label := variant.Name + "-" + strings.ReplaceAll(targetName, "/", "-")
					if err := saveFinding(activeRunner, *output, currentSeed, label, variant.Source, comparison, *minimizeFindings); err != nil {
						fatalSeed(currentSeed, err)
					}
					if *stopAfter > 0 && found >= *stopAfter {
						stop = true
						break
					}
				}
			}
			if stop {
				break
			}
		}
		if stop {
			break
		}
		if *progress > 0 && (index+1)%*progress == 0 {
			fmt.Printf("tested=%d findings=%d rate=%.1f programs/s\n", index+1, found, float64(index+1)/time.Since(started).Seconds())
		}
	}
	fmt.Printf("complete findings=%d elapsed=%s\n", found, time.Since(started).Round(time.Millisecond))
}

func saveFinding(runner difftest.Runner, root string, seed uint64, label string, source []byte, comparison difftest.Comparison, minimize bool) error {
	dir := filepath.Join(root, fmt.Sprintf("seed-%016x-%s", seed, label))
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
	report := fmt.Sprintf("seed: %d\ncase: %s\ntarget: %s\nsignature: %s\nhost exit: %d\nhost output: %q\nrenvo exit: %d\nrenvo output: %q\nrenvo diagnostic: %s\n", seed, label, runner.Target, final.Signature, final.Host.ExitCode, final.Host.Output, final.Renvo.ExitCode, final.Renvo.Output, final.Renvo.Diagnostic)
	if err := os.WriteFile(filepath.Join(dir, "report.txt"), []byte(report), 0o644); err != nil {
		return err
	}
	fmt.Printf("FOUND seed=%d signature=%s source=%s\n", seed, signature, filepath.Join(dir, "minimized.go"))
	return nil
}

func equivalentExecution(left, right difftest.Execution) bool {
	return left.Compiled == right.Compiled && left.Ran == right.Ran && left.TimedOut == right.TimedOut && left.ExitCode == right.ExitCode && bytes.Equal(left.Output, right.Output)
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
