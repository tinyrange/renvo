package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"j5.nz/rtg/target"
)

type artifactLoader func(string, target.ELFArtifactOptions) (target.Artifact, error)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, target.ArtifactFromELF))
}

func run(args []string, stdout io.Writer, stderr io.Writer, load artifactLoader) int {
	flags := flag.NewFlagSet("rtgboard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	boardName := flags.String("board", "ch32v003", "freestanding board profile")
	vector := flags.String("vector", "", "linked vector-table symbol (defaults to the board contract)")
	heap := flags.Uint64("heap", 0, "reserved heap bytes")
	stack := flags.Uint64("stack", 0, "reserved stack bytes (zero uses the board default)")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "usage: rtgboard [options] firmware.elf")
		fmt.Fprintln(flags.Output(), "validate a linked freestanding image before flashing")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}
	composition, ok := boardByName(*boardName)
	if !ok {
		fmt.Fprintf(stderr, "rtgboard: unsupported board %q\n", *boardName)
		return 2
	}
	vectorSymbol := *vector
	if vectorSymbol == "" {
		vectorSymbol = composition.Board.Startup.VectorSymbol
	}
	if vectorSymbol != composition.Board.Startup.VectorSymbol {
		fmt.Fprintf(stderr, "rtgboard: vector symbol %q does not match board contract %q\n", vectorSymbol, composition.Board.Startup.VectorSymbol)
		return 2
	}
	artifact, err := load(flags.Arg(0), target.ELFArtifactOptions{
		VectorSymbol: vectorSymbol,
		HeapSize:     *heap,
		StackSize:    *stack,
	})
	if err != nil {
		fmt.Fprintf(stderr, "rtgboard: inspect artifact: %v\n", err)
		return 1
	}
	validation := target.Validate(composition, artifact)
	writeReport(stdout, composition, artifact, validation)
	if !validation.OK() {
		for _, violation := range validation.Violations {
			fmt.Fprintf(stderr, "rtgboard: %s\n", violation.Error())
		}
		return 1
	}
	return 0
}

func boardByName(name string) (target.Composition, bool) {
	if name == "ch32v003" || name == "wch-ch32v003" {
		return target.CH32V003(), true
	}
	return target.Composition{}, false
}

func writeReport(out io.Writer, composition target.Composition, artifact target.Artifact, validation target.Validation) {
	fmt.Fprintf(out, "board=%s object=%s isa=%s abi=%s format=%s\n",
		composition.Board.Name, composition.Object.Name, composition.Object.ISA, composition.Object.ABI, composition.Object.Format.Name)
	sections := append([]target.Section(nil), artifact.Sections...)
	sort.Slice(sections, func(i int, j int) bool {
		if sections[i].Address != sections[j].Address {
			return sections[i].Address < sections[j].Address
		}
		return sections[i].Name < sections[j].Name
	})
	for _, section := range sections {
		if section.Flags&target.SectionAlloc == 0 || section.Size == 0 {
			continue
		}
		region := "flash"
		if section.Flags&target.SectionWrite != 0 {
			region = "ram"
		}
		fmt.Fprintf(out, "section=%s region=%s address=%#x size=%d load_address=%#x load_size=%d\n",
			section.Name, region, section.Address, section.Size, section.LoadAddress, section.LoadSize)
	}
	u := validation.Usage
	fmt.Fprintf(out, "flash used=%d free=%d capacity=%d\n", u.FlashUsed, u.FlashFree, u.FlashCapacity)
	fmt.Fprintf(out, "ram static=%d heap=%d stack=%d guard=%d free=%d capacity=%d\n",
		u.RAMStatic, u.HeapReserved, u.StackReserved, u.GuardReserved, u.RAMFree, u.RAMCapacity)
}
