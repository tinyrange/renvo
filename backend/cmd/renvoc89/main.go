package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"renvo.dev/backend/target"
)

type stringList []string

func (values *stringList) String() string {
	return fmt.Sprint([]string(*values))
}

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("renvoc89", flag.ContinueOnError)
	flags.SetOutput(stderr)
	mode := flags.String("mode", "automatic", "machine profile mode: automatic or explicit")
	name := flags.String("name", "c89", "profile name embedded in generated C")
	abi := flags.String("abi", "portable-c", "target ABI/toolchain contract name")
	endianName := flags.String("endian", "little", "target byte order: little or big")
	hosted := flags.Bool("hosted", false, "declare a hosted rather than freestanding C environment")
	intBits := flags.Int("int", 0, "RENVO language int width for explicit mode: 16, 32, or 64")
	pointerBits := flags.Int("pointer", 0, "data pointer width for explicit mode: 16, 32, or 64")
	preambleOnly := flags.Bool("preamble-only", false, "omit defined-arithmetic helper functions")
	output := flags.String("o", "-", "output C path, or - for stdout")
	var runtimeOps stringList
	flags.Var(&runtimeOps, "runtime", "available runtime operation (repeatable)")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "usage: renvoc89 [options]")
		fmt.Fprintln(flags.Output(), "emit a deterministic strict-C89 machine contract and support unit")
		flags.PrintDefaults()
	}
	if len(args) == 0 {
		flags.SetOutput(stdout)
		flags.Usage()
		return 0
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "renvoc89: unexpected argument %q\n", flags.Arg(0))
		return 2
	}
	endian, ok := parseEndian(*endianName)
	if !ok {
		fmt.Fprintf(stderr, "renvoc89: invalid endianness %q; want little or big\n", *endianName)
		return 2
	}
	var profile target.CMachineProfile
	switch *mode {
	case "automatic":
		if *intBits != 0 || *pointerBits != 0 {
			fmt.Fprintln(stderr, "renvoc89: -int and -pointer are only valid in explicit mode")
			return 2
		}
		profile = target.C89AutomaticProfile(*name, *hosted, endian, *abi, runtimeOps...)
	case "explicit":
		if *intBits == 0 || *pointerBits == 0 {
			fmt.Fprintln(stderr, "renvoc89: explicit mode requires -int and -pointer")
			return 2
		}
		profile = target.C89ExplicitProfile(*name, *hosted, *intBits, *pointerBits, endian, *abi, runtimeOps...)
	default:
		fmt.Fprintf(stderr, "renvoc89: invalid mode %q; want automatic or explicit\n", *mode)
		return 2
	}
	var source []byte
	var err error
	if *preambleOnly {
		source, err = profile.RenderC89Preamble()
	} else {
		source, err = profile.RenderC89Support()
	}
	if err != nil {
		fmt.Fprintf(stderr, "renvoc89: %v\n", err)
		return 2
	}
	if *output == "-" {
		if _, err := stdout.Write(source); err != nil {
			fmt.Fprintf(stderr, "renvoc89: write stdout: %v\n", err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*output, source, 0o644); err != nil {
		fmt.Fprintf(stderr, "renvoc89: write %s: %v\n", *output, err)
		return 1
	}
	return 0
}

func parseEndian(value string) (target.CEndian, bool) {
	if value == "little" {
		return target.CEndianLittle, true
	}
	if value == "big" {
		return target.CEndianBig, true
	}
	return "", false
}
