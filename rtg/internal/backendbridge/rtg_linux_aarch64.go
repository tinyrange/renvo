//go:build rtg && linux && (aarch64 || arm64)

package backendbridge

const rtgLinuxAarch64Sigchld = 17

func CompileUnitToOutputStripEnv(unit []byte, targetName string, outputPath string, stripSymbols bool, env []string) bool {
	backend := backendPath(env)
	if backend == "" || outputPath == "" || targetName == "" {
		return false
	}
	pid := syscall(172)
	if pid < 0 {
		return false
	}
	unitPath := tempPath("/tmp/rtg_unit_", pid, ".rtgu")
	scriptPath := tempPath("/tmp/rtg_backend_", pid, ".sh")
	if !writeFile(unitPath, unit, 420) {
		return false
	}
	script := backendScript(backend, targetName, outputPath, unitPath, stripSymbols)
	if !writeFile(scriptPath, script, 493) {
		return false
	}
	child := syscall(220, rtgLinuxAarch64Sigchld, 0, 0, 0, 0)
	if child == 0 {
		syscall(221, cstring(scriptPath), 0, 0)
		syscall(93, 111)
		return false
	}
	if child < 0 {
		return false
	}
	status := make([]int, 1)
	waited := syscall(260, child, status, 0, 0)
	if waited != child || status[0] != 0 {
		return false
	}
	return chmodOutput(outputPath)
}
