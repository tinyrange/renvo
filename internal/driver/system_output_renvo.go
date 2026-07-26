//go:build renvo

package driver

import (
	"renvo.dev/internal/backendbridge"
	"renvo.dev/std/os"
)

func compileSystemOutput(unit []byte, backendTarget string, virtualTarget string, output string, strip bool, windowsGUI bool, emitImage bool, arenaSize int, moduleLicense string, systemName string, binaryLimit int) (bool, Diagnostic) {
	binary, ok := backendbridge.CompileUnitToBytes(unit, backendTarget, strip, windowsGUI, emitImage, arenaSize, moduleLicense)
	if !ok || len(binary) == 0 {
		return false, Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-001", Message: "backend compilation failed"}
	}
	if virtualTarget == "browser/wasm32" && !emitImage {
		binary = PackageBrowserHTML(binary)
	}
	if len(binary) > binaryLimit {
		return false, systemBinaryLimitDiagnostic(systemName, len(binary), binaryLimit)
	}
	if output == "-" {
		print(string(binary))
		return true, Diagnostic{}
	}
	mode := 0755
	if emitImage || virtualTarget == "browser/wasm32" {
		mode = 0644
	}
	if os.WriteFile(output, binary, mode) != nil {
		return false, Diagnostic{Phase: "output", Code: "RENVO-OUTPUT-001", Message: "failed to write output"}
	}
	return true, Diagnostic{}
}
