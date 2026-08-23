//go:build renvo_wasi_backend

package main

// This command is compiled by Renvo, not Go. It is the fixed WASI backend
// module loaded after the compact frontend has produced a canonical unit.

var renvoDefaultTarget int = renvoTargetWasiWasm32
var renvoFixedTarget int = renvoTargetWasiWasm32
var renvoCompilerStripSymbols bool
var renvoKernelRelease string
var renvoKernelModuleName string
var renvoKernelBTF []byte
var renvoKernelSymvers []byte
var renvoKernelVersion string
var renvoKernelModuleSize int
var renvoKernelModuleNameOff int
var renvoKernelModuleInitOff int
var renvoKernelModuleExitOff int
var renvoKernelLicense string

func renvoPrintErr(text string) {
	write(2, []byte(text), -1)
}

func renvoPrintIntErr(value int) {
	if value == 0 {
		renvoPrintErr("0")
		return
	}
	if value < 0 {
		renvoPrintErr("-")
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		write(2, digits[i:i+1], -1)
	}
}

func renvoWasiBackendUsage() {
	renvoPrintErr("usage: renvo-wasi-backend [-s] [-arena-size bytes] -o output.wasm input.unit\n")
}

func renvoWasiBackendDecimal(text string) (int, bool) {
	if len(text) == 0 {
		return 0, false
	}
	value := 0
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
		value = value*10 + int(text[i]-'0')
		if value > 1073741824 {
			return 0, false
		}
	}
	return value, value == 0 || value >= 256
}

func renvoWasiBackendCompile(unit []byte, output int, arenaSize int) int {
	if !renvoUnitBindingMatchesTarget(unit, renvoTargetWasiWasm32) {
		renvoPrintErr("renvo: unit target does not match wasi/wasm32\n")
		return 1
	}
	prog, isUnit, ok := renvoDecodeUnitProgram(unit)
	if !isUnit || !ok {
		renvoPrintErr("renvo: invalid canonical unit\n")
		return 1
	}
	renvoSetTarget(renvoTargetWasiWasm32)
	context := renvoNewCompileContext(renvoTargetWasiWasm32, renvoCompilerStripSymbols, false, false)
	renvoConfigureCompileContext(context, "wasi/wasm32", "renvo", "")
	prog.c = *context
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !meta.ok {
		renvoPrintErr("renvo: backend metadata failed\n")
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(renvoTargetWasiWasm32, arenaSize)
	result := renvoTryCompileScalarProgramWasm32(&prog, &meta)
	if !result.ok {
		renvoPrintErr("renvo: WASI backend compilation failed\n")
		return 1
	}
	write(output, renvoCompileOutputDataWithContext(context, result.data, renvoTargetWasiWasm32), -1)
	return 0
}

func appMain(args []string, env []string) int {
	_ = env
	inputPath := ""
	outputPath := ""
	arenaSize := 0
	for i := 1; i < len(args); i++ {
		if args[i] == "-s" {
			renvoCompilerStripSymbols = true
			continue
		}
		if args[i] == "-o" && i+1 < len(args) {
			i++
			outputPath = args[i]
			continue
		}
		if args[i] == "-t" && i+1 < len(args) {
			i++
			if args[i] != "wasi/wasm32" {
				renvoPrintErr("renvo: fixed backend only supports wasi/wasm32\n")
				return 1
			}
			continue
		}
		if args[i] == "-arena-size" && i+1 < len(args) {
			i++
			var ok bool
			arenaSize, ok = renvoWasiBackendDecimal(args[i])
			if !ok {
				renvoPrintErr("renvo: invalid arena size\n")
				return 1
			}
			continue
		}
		if len(args[i]) > 0 && args[i][0] == '-' {
			renvoWasiBackendUsage()
			return 1
		}
		if inputPath != "" {
			renvoWasiBackendUsage()
			return 1
		}
		inputPath = args[i]
	}
	if inputPath == "" || outputPath == "" {
		renvoWasiBackendUsage()
		return 1
	}
	input := open(inputPath, 0)
	if input < 0 {
		renvoPrintErr("renvo: could not open canonical unit\n")
		return 1
	}
	var unit []byte
	unit = renvoReadAll(input, unit)
	close(input)
	output := open(outputPath, 578)
	if output < 0 {
		renvoPrintErr("renvo: could not open backend output\n")
		return 1
	}
	status := renvoWasiBackendCompile(unit, output, arenaSize)
	if status == 0 {
		chmod(output, 0755)
	}
	close(output)
	return status
}
