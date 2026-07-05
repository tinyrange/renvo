//go:build rtg && !(linux && (amd64 || aarch64 || arm64))

package backendbridge

import rtgx "j5.nz/rtg"

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, env []string) bool {
	_ = env
	return rtgx.RtgCompileUnitToOutputStrip(unit, targetName, outputPath, stripSymbols)
}
