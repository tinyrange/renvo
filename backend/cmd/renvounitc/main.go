package main

import (
	"flag"
	"fmt"
	"os"

	"renvo.dev/backend/unit"
)

func main() {
	output := flag.String("o", "-", "output unit path")
	dump := flag.Bool("dump", false, "decode and print canonical source instead of encoding")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: renvounitc [-o output.unit] input.go...\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *dump {
		if flag.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "renvounitc: -dump expects exactly one unit input")
			os.Exit(1)
		}
		program, err := unit.ReadFile(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "renvounitc: failed to read unit: %v\n", err)
			os.Exit(1)
		}
		if _, err := os.Stdout.Write(unit.Source(program)); err != nil {
			fmt.Fprintf(os.Stderr, "renvounitc: failed to write source: %v\n", err)
			os.Exit(1)
		}
		return
	}

	program, err := unit.ConvertFiles(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "renvounitc: conversion failed: %v\n", err)
		os.Exit(1)
	}
	if err := unit.WriteFile(*output, program); err != nil {
		fmt.Fprintf(os.Stderr, "renvounitc: failed to write unit: %v\n", err)
		os.Exit(1)
	}
}
