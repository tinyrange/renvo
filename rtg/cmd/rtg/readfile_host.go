//go:build !rtg

package main

import "os"

func readUnitFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
