//go:build rtg

package runtime

func ArenaMark() int {
	return 0
}

func ArenaReset(mark int) {
}

func ArenaPersistMark() int {
	return 0
}

func ArenaPersistReset(mark int) {
}

func ArenaPersistString(value string) string {
	return value
}

func ArenaPersistBytes(value []byte) []byte {
	return value
}
