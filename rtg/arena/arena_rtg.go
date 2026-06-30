//go:build rtg

package arena

import "runtime"

func Mark() int {
	if runtime.GOARCH != "amd64" {
		return 0
	}
	return runtime.ArenaMark()
}

func Reset(mark int) {
	if runtime.GOARCH != "amd64" {
		return
	}
	runtime.ArenaReset(mark)
}

func PersistMark() int {
	if runtime.GOARCH != "amd64" {
		return 0
	}
	return runtime.ArenaPersistMark()
}

func PersistReset(mark int) {
	if runtime.GOARCH != "amd64" {
		return
	}
	runtime.ArenaPersistReset(mark)
}

func PersistString(value string) string {
	if runtime.GOARCH != "amd64" {
		return value
	}
	return runtime.ArenaPersistString(value)
}

func PersistBytes(value []byte) []byte {
	if runtime.GOARCH != "amd64" {
		return value
	}
	return runtime.ArenaPersistBytes(value)
}
