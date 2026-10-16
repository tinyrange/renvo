//go:build !renvo && (!linux || !amd64 || faketime)

package systemclock

func nativeNow() uint64 { return 0 }
