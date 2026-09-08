// Command renvoperf gates complete-compiler self-hosting for every Tier 1.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"renvo.dev/internal/perfgate"
)

func main() {
	root := flag.String("root", ".", "candidate checkout")
	reference := flag.String("reference", "", "checkout of the policy's reference revision (default: temporary git worktree)")
	target := flag.String("target", perfgate.NativeTarget(), "Tier 1 compiler execution target")
	reportPath := flag.String("report", "sandbox/performance/report.json", "JSON report, including failures")
	matrix := flag.Bool("matrix", false, "print required CI matrix from the shared policy")
	platform := flag.String("platform", "", "matrix platform: linux, windows, darwin, or virtual")
	printReference := flag.Bool("reference-revision", false, "print the reference commit from the policy")
	vmRequest := flag.String("execute-vm", "", "internal VM runner request")
	flag.Parse()
	if *vmRequest != "" {
		if err := perfgate.ExecuteVM(*vmRequest); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	p := perfgate.Load()
	if err := p.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *printReference {
		fmt.Println(p.ReferenceRevision)
		return
	}
	if *matrix {
		targets := p.Targets
		if *platform != "" {
			var err error
			targets, err = p.PlatformTargets(*platform)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Include []perfgate.Target `json:"include"`
		}{targets})
		return
	}
	report, err := perfgate.Run(*root, *reference, *target, os.Stdout)
	if err != nil && len(report.Failures) == 0 {
		report.Failures = []string{err.Error()}
	}
	data, marshalErr := json.MarshalIndent(report, "", "  ")
	if marshalErr == nil {
		marshalErr = os.MkdirAll(filepath.Dir(*reportPath), 0755)
	}
	if marshalErr == nil {
		marshalErr = os.WriteFile(*reportPath, append(data, '\n'), 0644)
	}
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, "write report:", marshalErr)
		os.Exit(1)
	}
	fmt.Println("Performance report:", *reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("PASS: complete compiler self-hosting")
}
