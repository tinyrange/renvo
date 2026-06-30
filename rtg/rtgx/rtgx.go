package rtgx

import (
	"fmt"
	"os"
	"path/filepath"

	"j5.nz/rtg/rtg/arena"
	"j5.nz/rtg/rtg/link"
	targetpkg "j5.nz/rtg/rtg/target"
	"j5.nz/rtg/rtg/unit"
)

type Options struct {
	Target       string
	Output       string
	BackendRoot  string
	StripSymbols bool
}

type Artifact struct {
	Target             string
	Output             []byte
	LinkedSource       []byte
	LinkedUnits        []string
	ReachableFunctions []string
	Entrypoint         unit.Symbol
}

func CompileUnits(units []unit.Unit, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	data, err := CompileUnitsBytes(units, opts)
	if err != nil {
		return err
	}
	return writeOutput(data, opts.Output)
}

func CompileUnitsBytes(units []unit.Unit, opts Options) ([]byte, error) {
	plan, err := link.Build(units)
	if err != nil {
		return nil, err
	}
	linkedSource := link.Source(plan)
	return CompileSourceBytes(linkedSource, opts)
}

func CompileUnitsArtifact(units []unit.Unit, opts Options) (Artifact, error) {
	plan, err := link.Build(units)
	if err != nil {
		return Artifact{}, err
	}
	linkedSource := link.Source(plan)
	compiled, err := CompileSourceArtifact(linkedSource, opts)
	if err != nil {
		return Artifact{}, err
	}
	linked := link.SourceArtifact(plan)
	compiled.LinkedSource = copyBytes(linkedSource)
	compiled.LinkedUnits = copyStrings(linked.LinkedUnits)
	compiled.ReachableFunctions = copyStrings(linked.ReachableFunctions)
	compiled.Entrypoint = linked.Entrypoint
	return compiled, nil
}

func copyBytes(values []byte) []byte {
	out := make([]byte, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
}

func copyStrings(values []string) []string {
	out := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
}

func CompileUnitSources(sources []unit.SourceFile, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	data, err := CompileUnitSourcesBytes(sources, opts)
	if err != nil {
		return err
	}
	return writeOutput(data, opts.Output)
}

func CompileUnitSourcesBytes(sources []unit.SourceFile, opts Options) ([]byte, error) {
	units, err := unit.ParseSources(sources)
	if err != nil {
		return nil, err
	}
	return CompileUnitsBytes(units, opts)
}

func CompileTextUnitSources(sources []unit.TextSourceFile, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	data, err := CompileTextUnitSourcesBytes(sources, opts)
	if err != nil {
		return err
	}
	return writeOutput(data, opts.Output)
}

func CompileTextUnitSourcesReset(sources []unit.TextSourceFile, opts Options, resetMark int) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	linkedSource, err := LinkTextUnitSourcesReset(sources, resetMark)
	if err != nil {
		return err
	}
	return CompileSourceToOutput(linkedSource, opts)
}

func CompileTextUnitSourcesBytes(sources []unit.TextSourceFile, opts Options) ([]byte, error) {
	return CompileTextUnitSourcesBytesReset(sources, opts, -1)
}

func CompileTextUnitSourcesBytesReset(sources []unit.TextSourceFile, opts Options, resetMark int) ([]byte, error) {
	linkedSource, err := LinkTextUnitSourcesReset(sources, resetMark)
	if err != nil {
		return nil, err
	}
	return CompileSourceBytes(linkedSource, opts)
}

func LinkTextUnitSourcesReset(sources []unit.TextSourceFile, resetMark int) ([]byte, error) {
	mark := arena.Mark()
	units, err := unit.ParseTextSources(sources)
	if err != nil {
		arena.Reset(mark)
		if resetMark >= 0 {
			arena.Reset(resetMark)
		}
		return nil, err
	}
	plan, err := link.Build(units)
	if err != nil {
		arena.Reset(mark)
		if resetMark >= 0 {
			arena.Reset(resetMark)
		}
		return nil, err
	}
	linkedSource := arena.PersistBytes(link.Source(plan))
	arena.Reset(mark)
	if resetMark >= 0 {
		arena.Reset(resetMark)
	}
	return linkedSource, nil
}

func CompileUnitSourcesArtifact(sources []unit.SourceFile, opts Options) (Artifact, error) {
	units, err := unit.ParseSources(sources)
	if err != nil {
		return Artifact{}, err
	}
	return CompileUnitsArtifact(units, opts)
}

func CompileSource(source []byte, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	data, err := CompileSourceBytes(source, opts)
	if err != nil {
		return err
	}
	return writeOutput(data, opts.Output)
}

func CompileSourceBytes(source []byte, opts Options) ([]byte, error) {
	target := opts.Target
	if target == "" {
		target = targetpkg.Default()
	}
	if !targetpkg.Supported(target) {
		return nil, fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", target, targetpkg.List())
	}
	data, err := compileSourceToBytes(source, target, opts.BackendRoot, opts.StripSymbols)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func CompileSourceToOutput(source []byte, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	target := opts.Target
	if target == "" {
		target = targetpkg.Default()
	}
	if !targetpkg.Supported(target) {
		return fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", target, targetpkg.List())
	}
	return compileSourceToOutput(source, target, opts.BackendRoot, opts.StripSymbols, opts.Output)
}

func CompileSourceTextBytes(source string, opts Options) ([]byte, error) {
	return CompileSourceBytes([]byte(source), opts)
}

func CompileSourceArtifact(source []byte, opts Options) (Artifact, error) {
	data, err := CompileSourceBytes(source, opts)
	if err != nil {
		return Artifact{}, err
	}
	target := opts.Target
	if target == "" {
		target = targetpkg.Default()
	}
	return Artifact{Target: target, Output: data}, nil
}

func writeOutput(data []byte, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	if outputPath == "-" {
		fmt.Fprint(os.Stdout, bytesToString(data))
		return nil
	}
	output, err := filepath.Abs(outputPath)
	if err != nil {
		return err
	}
	return os.WriteFile(output, data, 493)
}

func bytesToString(data []byte) string {
	var out []byte
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return string(out)
}
