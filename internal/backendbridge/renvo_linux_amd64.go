//go:build renvo && linux && amd64

package backendbridge

import renvo "renvo.dev/backend"

func InitializeObjectCache(targetName string) { renvo.RenvoInitializeObjectCache(targetName) }

type CompileSession struct{ inner *renvo.RenvoCompileSession }

func BeginCompileSession(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, arenaSize int, moduleLicense string, objectFile bool) *CompileSession {
	return &CompileSession{inner: renvo.RenvoBeginCompileSession(unit, targetName, outputPath, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI, ModuleLicense: moduleLicense, ObjectFile: objectFile})}
}

func (s *CompileSession) Step() bool { return s == nil || s.inner == nil || s.inner.Step() }
func (s *CompileSession) Result() bool {
	return s != nil && s.inner != nil && s.inner.Result()
}

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, emitImage bool, arenaSize int, moduleLicense string, objectFile bool, args []string, env []string) bool {
	_ = args
	_ = env
	return renvo.RenvoCompileUnitToOutputWithOptions(unit, targetName, outputPath, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI, EmitImage: emitImage, ModuleLicense: moduleLicense, ObjectFile: objectFile})
}

func CompileUnitToImage(unit []byte, targetName string, stripSymbols bool, arenaSize int, moduleLicense string) ([]byte, bool) {
	return CompileUnitToBytes(unit, targetName, stripSymbols, false, true, arenaSize, moduleLicense)
}

func CompileUnitToBytes(unit []byte, targetName string, stripSymbols bool, windowsGUI bool, emitImage bool, arenaSize int, moduleLicense string) ([]byte, bool) {
	return renvo.RenvoCompileUnitToBytesWithOptions(unit, targetName, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI, EmitImage: emitImage, ModuleLicense: moduleLicense})
}
