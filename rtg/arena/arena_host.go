//go:build !rtg

package arena

func Mark() int {
	return 0
}

func Reset(mark int) {
}

func PersistMark() int {
	return 0
}

func PersistReset(mark int) {
}

func PersistString(value string) string {
	return value
}

func PersistBytes(value []byte) []byte {
	return value
}
