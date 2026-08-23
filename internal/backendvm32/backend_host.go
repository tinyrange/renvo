//go:build !renvo

package backendvm32

import (
	"renvo.dev/internal/driver"
	"renvo.dev/internal/targetinfo"
	internalunit "renvo.dev/internal/unit"
)

const Target = "vm/vm32"

// SeedVersion changes whenever the fixed VM32 bootstrap boundary changes in a
// way that invalidates derived compiler artifacts.
const SeedVersion = 1

// Backend is the native unit-to-VM32 seed. It deliberately accepts exactly
// one target; target-native backends are derived artifacts rather than part of
// this package.
type Backend struct{}

func (Backend) CompileUnit(unit []byte, target string, strip bool, windowsGUI bool) driver.BackendResult {
	return compile(unit, driver.BackendCompileOptions{
		Target: target, Strip: strip, WindowsGUI: windowsGUI,
	})
}

func (Backend) CompileUnitWithArena(unit []byte, target string, strip bool, windowsGUI bool, arenaSize int) driver.BackendResult {
	return compile(unit, driver.BackendCompileOptions{
		Target: target, Strip: strip, WindowsGUI: windowsGUI, ArenaSize: arenaSize,
	})
}

func (Backend) CompileUnitWithOptions(unit []byte, options driver.BackendCompileOptions) driver.BackendResult {
	return compile(unit, options)
}

func compile(unit []byte, options driver.BackendCompileOptions) driver.BackendResult {
	if options.Target != Target || options.Mode != "" && options.Mode != driver.ModeExecutable ||
		options.ObjectFile || options.WindowsGUI {
		return seedFailure("VM32 seed backend only accepts vm/vm32 executable output")
	}
	if !unitBindingMatches(unit) {
		return seedFailure("unit target binding does not match the VM32 seed backend")
	}
	data, ok := RenvoCompileUnitToBytesWithOptions(unit, Target, RenvoCompileOptions{
		ArenaSize: options.ArenaSize, StripSymbols: options.Strip,
		EmitImage: options.EmitImage, ModuleNamePath: options.Output,
	})
	if !ok {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-001", Message: "VM32 seed backend compilation failed",
		}}
	}
	return driver.BackendResult{Binary: data, Ok: true}
}

func unitBindingMatches(data []byte) bool {
	binding, bound := internalunit.ReadTargetBinding(data)
	descriptor, known := targetinfo.Lookup(Target)
	return bound && known && binding.Target == descriptor.Name &&
		binding.Definition == string(descriptor.Definition[:]) &&
		binding.DescriptorVersion == descriptor.DescriptorVersion
}

func seedFailure(message string) driver.BackendResult {
	return driver.BackendResult{Diagnostic: driver.Diagnostic{
		Phase: "backend", Code: "RENVO-BACKEND-007", Message: message,
	}}
}
