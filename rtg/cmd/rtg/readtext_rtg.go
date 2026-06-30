//go:build rtg

package main

import (
	"os"

	"j5.nz/rtg/rtg/arena"
)

func readUnitTextFile(path string) (string, error) {
	mark := arena.Mark()
	data, err := os.ReadFile(path)
	if err != nil {
		arena.Reset(mark)
		return "", err
	}
	text := arena.PersistString(string(data))
	arena.Reset(mark)
	return text, nil
}
