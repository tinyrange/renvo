//go:build renvo && linux && (aarch64 || arm64)

package backendbridge

import renvo "renvo.dev/backend"

func InitializeObjectCache(targetName string) { renvo.RenvoInitializeObjectCache(targetName) }
func TargetSupported(targetName string) bool  { return renvo.RenvoTargetSupported(targetName) }
func TargetBinding(targetName string) (string, string, int, bool) {
	return renvo.RenvoTargetBinding(targetName)
}
func TargetHasBuildTag(targetName string, tag string) bool {
	return renvo.RenvoTargetHasBuildTag(targetName, tag)
}
func TargetHasCapability(targetName string, capability string) bool {
	return renvo.RenvoTargetHasCapability(targetName, capability)
}

type CompileSession struct{ inner *renvo.RenvoCompileSession }

func BeginCompileSession(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, arenaSize int, moduleLicense string, objectFile bool, code16 bool, regParm int) *CompileSession {
	return &CompileSession{inner: renvo.RenvoBeginCompileSession(unit, targetName, outputPath, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI, ModuleLicense: moduleLicense, ObjectFile: objectFile, Code16: code16, RegParm: regParm})}
}

func (s *CompileSession) Step() bool { return s == nil || s.inner == nil || s.inner.Step() }
func (s *CompileSession) Result() bool {
	return s != nil && s.inner != nil && s.inner.Result()
}

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool, emitImage bool, arenaSize int, moduleLicense string, objectFile bool, code16 bool, regParm int, args []string, env []string) bool {
	_ = args
	_ = env
	return renvo.RenvoCompileUnitToOutputWithOptions(unit, targetName, outputPath, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI, EmitImage: emitImage, ModuleLicense: moduleLicense, ObjectFile: objectFile, Code16: code16, RegParm: regParm})
}

func CompileUnitToImage(unit []byte, targetName string, stripSymbols bool, arenaSize int, moduleLicense string) ([]byte, bool) {
	return CompileUnitToBytes(unit, targetName, stripSymbols, false, true, arenaSize, moduleLicense)
}

func CompileUnitToBytes(unit []byte, targetName string, stripSymbols bool, windowsGUI bool, emitImage bool, arenaSize int, moduleLicense string) ([]byte, bool) {
	return renvo.RenvoCompileUnitToBytesWithOptions(unit, targetName, renvo.RenvoCompileOptions{ArenaSize: arenaSize, StripSymbols: stripSymbols, WindowsGUI: windowsGUI, EmitImage: emitImage, ModuleLicense: moduleLicense})
}
