//go:build rtg

package main

import (
	"os"

	"j5.nz/rtg/rtg/arena"
)

func readUnitFile(path string) ([]byte, error) {
	mark := arena.Mark()
	data, err := os.ReadFile(path)
	if err != nil {
		arena.Reset(mark)
		return nil, err
	}
	data = arena.PersistBytes(data)
	arena.Reset(mark)
	return data, nil
}
