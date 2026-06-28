package rtgx

import (
	"fmt"
	"os"
	"path/filepath"

	"j5.nz/rtg/rtg/link"
	targetpkg "j5.nz/rtg/rtg/target"
	"j5.nz/rtg/rtg/unit"
)

type Options struct {
	Target      string
	Output      string
	BackendRoot string
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
	artifact, err := CompileUnitsArtifact(units, opts)
	if err != nil {
		return err
	}
	return writeOutput(artifact.Output, opts.Output)
}

func CompileUnitsBytes(units []unit.Unit, opts Options) ([]byte, error) {
	artifact, err := CompileUnitsArtifact(units, opts)
	if err != nil {
		return nil, err
	}
	return artifact.Output, nil
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
	artifact, err := CompileUnitSourcesArtifact(sources, opts)
	if err != nil {
		return err
	}
	return writeOutput(artifact.Output, opts.Output)
}

func CompileUnitSourcesBytes(sources []unit.SourceFile, opts Options) ([]byte, error) {
	artifact, err := CompileUnitSourcesArtifact(sources, opts)
	if err != nil {
		return nil, err
	}
	return artifact.Output, nil
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
	artifact, err := CompileSourceArtifact(source, opts)
	if err != nil {
		return err
	}
	return writeOutput(artifact.Output, opts.Output)
}

func CompileSourceBytes(source []byte, opts Options) ([]byte, error) {
	artifact, err := CompileSourceArtifact(source, opts)
	if err != nil {
		return nil, err
	}
	return artifact.Output, nil
}

func CompileSourceArtifact(source []byte, opts Options) (Artifact, error) {
	target := opts.Target
	if target == "" {
		target = targetpkg.Default()
	}
	if !targetpkg.Supported(target) {
		return Artifact{}, fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", target, targetpkg.List())
	}
	data, err := compileSourceToBytes(source, target, opts.BackendRoot)
	if err != nil {
		return Artifact{}, err
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
