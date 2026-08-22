//go:build !renvo

package backendcompiled

import (
	"renvo.dev/internal/driver"
	"renvo.dev/internal/targetinfo"
	internalunit "renvo.dev/internal/unit"
)

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
	target := options.Target
	if options.Mode == driver.ModeKernelModule {
		target = "linux-kernel/amd64"
	}
	if !unitBindingMatches(unit, target) {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-007",
			Message: "unit target binding does not match the built-in backend",
		}}
	}
	data, ok := RenvoCompileUnitToBytesWithOptions(unit, target, RenvoCompileOptions{
		ArenaSize: options.ArenaSize, StripSymbols: options.Strip,
		WindowsGUI: options.WindowsGUI, EmitImage: options.EmitImage,
		ModuleLicense: options.ModuleLicense, ModuleNamePath: options.Output,
		ObjectFile: options.ObjectFile || options.Mode == driver.ModeObject,
	})
	if !ok {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-001", Message: "built-in backend compilation failed",
		}}
	}
	return driver.BackendResult{Binary: data, Ok: true}
}

func unitBindingMatches(data []byte, target string) bool {
	binding, bound := internalunit.ReadTargetBinding(data)
	descriptor, known := targetinfo.Lookup(target)
	return bound && known && binding.Target == descriptor.Name &&
		binding.Definition == string(descriptor.Definition[:]) &&
		binding.DescriptorVersion == descriptor.DescriptorVersion
}
