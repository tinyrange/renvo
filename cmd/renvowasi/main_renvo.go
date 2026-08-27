//go:build renvo_wasi_frontend

package main

import (
	"renvo.dev/internal/driver"
	"renvo.dev/internal/unit"
)

func renvoWasiHexDigit(value byte) int {
	if value >= '0' && value <= '9' {
		return int(value - '0')
	}
	if value >= 'a' && value <= 'f' {
		return int(value-'a') + 10
	}
	if value >= 'A' && value <= 'F' {
		return int(value-'A') + 10
	}
	return -1
}

func renvoWasiDefinition(value string) (string, bool) {
	if len(value) != 64 {
		return "", false
	}
	definition := make([]byte, 32)
	for i := 0; i < len(definition); i++ {
		high := renvoWasiHexDigit(value[i*2])
		low := renvoWasiHexDigit(value[i*2+1])
		if high < 0 || low < 0 {
			return "", false
		}
		definition[i] = byte(high<<4 | low)
	}
	return string(definition), true
}

func renvoWasiPositiveDecimal(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		result = result*10 + int(value[i]-'0')
	}
	return result, true
}

func writeOutput(path string, data []byte) bool {
	fd := open(path, 578)
	if fd < 0 {
		return false
	}
	written := 0
	for written < len(data) {
		n := write(fd, data[written:], -1)
		if n <= 0 {
			close(fd)
			return false
		}
		written += n
	}
	if chmod(fd, 0644) != 0 {
		close(fd)
		return false
	}
	return close(fd) == 0
}

func appMain(args []string, env []string) int {
	workDir := "."
	stdRoot := "./std"
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) > 4 && item[:4] == "PWD=" {
			workDir = item[4:]
		}
		if len(item) > 14 && item[:14] == "RENVO_STDROOT=" {
			stdRoot = item[14:]
		}
	}
	if renvoWasiCObjectCompileRequested(args) {
		return renvoWasiCCompile(args, workDir, stdRoot)
	}
	output := "a.unit"
	packageArg := "."
	target := "wasi/wasm32"
	definition := ""
	descriptorVersion := 0
	tags := make([]string, 0, 4)
	mode := driver.ModeExecutable
	cCompiler := len(args) > 1 && args[1] == "cc"
	for i := 1; i < len(args); i++ {
		if i == 1 && cCompiler {
			continue
		} else if args[i] == "-o" && i+1 < len(args) {
			output = args[i+1]
			i++
		} else if args[i] == "-tags" && i+1 < len(args) {
			tags = append(tags, args[i+1])
			i++
		} else if args[i] == "-t" && i+1 < len(args) {
			target = args[i+1]
			i++
		} else if args[i] == "-target-definition" && i+1 < len(args) {
			i++
			var ok bool
			definition, ok = renvoWasiDefinition(args[i])
			if !ok {
				print("renvo: invalid target definition\n")
				return 2
			}
		} else if args[i] == "-target-version" && i+1 < len(args) {
			i++
			var ok bool
			descriptorVersion, ok = renvoWasiPositiveDecimal(args[i])
			if !ok {
				print("renvo: invalid target descriptor version\n")
				return 2
			}
		} else if (args[i] == "-arena-size" || args[i] == "-module-license") && i+1 < len(args) {
			i++
		} else if args[i] == "-mode" && i+1 < len(args) {
			mode = args[i+1]
			i++
		} else if args[i] == "-mode=object" {
			mode = driver.ModeObject
		} else if args[i] == "-mode=executable" {
			mode = driver.ModeExecutable
		} else if args[i] == "-mode=kernel-module" {
			mode = driver.ModeKernelModule
		} else if args[i] == "-s" || args[i] == "-emit-unit" || args[i] == "-emit-image" || args[i] == "-windows-gui" {
			// These options affect the separately invoked backend only.
		} else if len(args[i]) > 0 && args[i][0] == '-' {
			print("renvo: unsupported browser frontend option ")
			print(args[i])
			print("\n")
			return 2
		} else {
			packageArg = args[i]
		}
	}
	if mode != driver.ModeExecutable && mode != driver.ModeObject && mode != driver.ModeKernelModule {
		print("renvo: unsupported build mode ")
		print(mode)
		print("\n")
		return 2
	}
	var built driver.PackageUnitResult
	if cCompiler {
		normalized := driver.NormalizeCCompilerCommand(args)
		full := driver.BuildFromFSOneShot(normalized[1:], workDir, stdRoot, driver.RenvoFS{})
		built = driver.PackageUnitResult{Unit: full.Unit, Path: full.ErrorPath, Ok: full.Ok, Phase: full.Error, Diagnostic: full.Diagnostic}
	} else {
		built = driver.BuildPackageUnitCompactMode(packageArg, target, tags, mode, workDir, stdRoot, driver.RenvoFS{})
	}
	if !built.Ok {
		print(driver.FormatDiagnostic(built.Diagnostic))
		return 1
	}
	if definition != "" {
		bound, ok := unit.BindTarget(built.Unit, unit.TargetBinding{
			Target: target, Definition: definition, DescriptorVersion: descriptorVersion,
		})
		if !ok {
			print("renvo: could not bind custom target\n")
			return 1
		}
		built.Unit = bound
	}
	if !writeOutput(output, built.Unit) {
		print("renvo: could not write output\n")
		return 1
	}
	return 0
}

func renvoWasiCCompile(args []string, workDir string, stdRoot string) int {
	normalized := driver.NormalizeCCompilerCommand(args)
	built := driver.BuildFromFSOneShot(normalized[1:], workDir, stdRoot, driver.RenvoFS{})
	if !built.Ok {
		print(driver.FormatDiagnostic(built.Diagnostic))
		return 1
	}
	if !writeOutput(built.Options.Output, built.Unit) {
		print("renvo cc: could not write frontend output\n")
		return 1
	}
	return 0
}

func renvoWasiCObjectCompileRequested(args []string) bool {
	if len(args) < 2 || args[1] != "cc" {
		return false
	}
	for i := 2; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "-mode=object" || args[i] == "-mode" && i+1 < len(args) && args[i+1] == "object" {
			return true
		}
	}
	return false
}
