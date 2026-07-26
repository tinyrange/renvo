//go:build !renvo

package main

import (
	"os"
	"path/filepath"

	"renvo.dev/internal/bootstrap"
	"renvo.dev/internal/driver"
)

func main() {
	os.Exit(bootstrap.Run(os.Args, os.Environ(), siblingBackend()))
}

func siblingBackend() driver.Backend {
	executable, err := os.Executable()
	if err != nil {
		return nil
	}
	name := "renvo-backend"
	if filepath.Ext(executable) == ".exe" {
		name += ".exe"
	}
	return driver.CommandBackend{Path: filepath.Join(filepath.Dir(executable), name)}
}
