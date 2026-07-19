//go:build !renvo

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"renvo.dev/internal/testfront"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var output string
	flags := flag.NewFlagSet("renvotest", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&output, "o", "", "generated package output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if output == "" {
		fmt.Fprintln(os.Stderr, "renvotest: missing -o output directory")
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "renvotest: expected exactly one package directory")
		return 2
	}
	result, err := testfront.GeneratePackage(flags.Arg(0))
	if err != nil {
		printGenerateError(err)
		return 1
	}
	if err := testfront.WritePackage(output, result); err != nil {
		fmt.Fprintf(os.Stderr, "renvotest: failed to write generated package: %v\n", err)
		return 1
	}
	return 0
}

func printGenerateError(err error) {
	if errors.Is(err, testfront.ErrNoTests) {
		fmt.Fprintln(os.Stderr, "renvotest: no tests found")
		return
	}
	if errors.Is(err, testfront.ErrExternalTests) {
		fmt.Fprintln(os.Stderr, "renvotest: external test packages are not supported yet")
		return
	}
	fmt.Fprintf(os.Stderr, "renvotest: failed to generate test runner: %v\n", err)
}
