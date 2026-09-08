//go:build !renvo

package backendjit

import (
	"fmt"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/linkedimage"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/unit"
	"renvo.dev/std/vm"
)

// NewPrepared executes an already prepared definition without reading host
// files. Preparation, imports, and execution are owned by the caller.
func NewPrepared(prepared Prepared, bootstrap driver.Backend, runner Runner) *Backend {
	return &Backend{prepared: prepared, bootstrap: bootstrap, runner: runner}
}

// MemoryRunner executes a VM32 prepared compiler in an isolated virtual
// filesystem. Neither compiler inputs nor generated outputs touch host files.
type MemoryRunner struct{}

func (MemoryRunner) Run(artifact rtgb.Artifact, request Request) driver.BackendResult {
	fail := func(message string) driver.BackendResult {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-009", Message: message,
		}}
	}
	if request.Protocol != ProtocolVersion || artifact.Protocol != ProtocolVersion ||
		artifact.Unit != unit.Version || artifact.Optimization != OptimizationVersion {
		return fail("prepared backend protocol is incompatible")
	}
	image, err := linkedimage.Decode(artifact.Payload)
	if err != nil || image.Target != "vm/vm32" || artifact.Host != image.Target {
		return fail("prepared backend requires a VM32 payload")
	}
	input := request.Unit
	if len(request.Source) != 0 {
		input = request.Source
	}
	args := []string{"renvo-prepared-backend", "-t", artifact.Descriptor.Name}
	o := request.Options
	if o.Strip {
		args = append(args, "-s")
	}
	if o.WindowsGUI {
		args = append(args, "-windows-gui")
	}
	if o.EmitImage {
		args = append(args, "-emit-image")
	}
	if o.ObjectFile || o.Mode == driver.ModeObject {
		args = append(args, "-object")
	}
	if o.ArenaSize > 0 {
		args = append(args, "-arena-size", decimal(o.ArenaSize))
	}
	if o.Output != "" {
		args = append(args, "-module-name", o.Output)
	}
	if o.ModuleLicense != "" {
		args = append(args, "-module-license", o.ModuleLicense)
	}
	args = append(args, "-o", "output.bin", "-")
	r := vm.RunConfig(image.Native, vm.Config{
		Limits: vm.Limits{Steps: 2_000_000_000, Memory: 128 << 20},
		Args:   args, Env: []string{"PWD=/"}, Stdin: input,
	})
	if r.Trap != vm.TrapNone || r.ExitCode != 0 {
		return fail(fmt.Sprintf("prepared compiler exit %d, trap %d at pc %d after %d steps: %s%s",
			r.ExitCode, r.Trap, r.TrapPC, r.Steps, r.Output, r.Stderr))
	}
	for _, file := range r.Files {
		if file.Name == "output.bin" && len(file.Data) != 0 {
			return driver.BackendResult{Binary: file.Data, Ok: true}
		}
	}
	return fail("prepared compiler produced no output")
}
