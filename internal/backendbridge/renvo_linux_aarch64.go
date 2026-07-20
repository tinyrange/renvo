//go:build renvo && linux && (aarch64 || arm64)

package backendbridge

import renvo "renvo.dev/backend"

func InitializeObjectCache(targetName string) { renvo.RenvoInitializeObjectCache(targetName) }

type CompileSession struct{ inner *renvo.RenvoCompileSession }

func BeginCompileSession(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, arenaSize int) *CompileSession {
	return &CompileSession{inner: renvo.RenvoBeginCompileSession(unit, targetName, outputPath, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI})}
}

func (s *CompileSession) Step() bool { return s == nil || s.inner == nil || s.inner.Step() }
func (s *CompileSession) Result() bool {
	return s != nil && s.inner != nil && s.inner.Result()
}

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, arenaSize int, args []string, env []string) bool {
	_ = args
	_ = env
	return renvo.RenvoCompileUnitToOutputWithOptions(unit, targetName, outputPath, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI})
}
