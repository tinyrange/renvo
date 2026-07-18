//go:build rtg && !(linux && (amd64 || aarch64 || arm64))

package backendbridge

import rtgx "j5.nz/rtg"

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, arenaSize int, args []string, env []string) bool {
	_ = args
	_ = env
	return rtgx.RtgCompileUnitToOutputWithOptions(unit, targetName, outputPath, rtgx.RtgCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI})
}
