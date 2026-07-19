//go:build !renvo

// Package process supplies the deliberately small process-launching surface
// needed by graphical development tools.
package process

import "os/exec"

func Start(path, directory string) bool {
	command := exec.Command(path)
	command.Dir = directory
	if command.Start() != nil {
		return false
	}
	if command.Process != nil {
		_ = command.Process.Release()
	}
	return true
}
