//go:build !renvo

package fontcache

func staticBytes(data string) []byte { return []byte(data) }
