//go:build renvo && !(linux && (amd64 || aarch64 || arm64))

package backendbridge

import renvo "renvo.dev/backend"

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, arenaSize int, args []string, env []string) bool {
	_ = args
	_ = env
	return renvo.RenvoCompileUnitToOutputWithOptions(unit, targetName, outputPath, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI})
}
