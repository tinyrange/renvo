//go:build rtg && linux && (aarch64 || arm64)

package backendbridge

import rtgx "j5.nz/rtg"

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, args []string, env []string) bool {
	_ = args
	_ = env
	return rtgx.RtgCompileUnitToOutputStrip(unit, targetName, outputPath, stripSymbols)
}
