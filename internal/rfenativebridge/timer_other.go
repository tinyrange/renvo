//go:build !renvo && (!cgo || !linux || !amd64)

package rfenativebridge

func MonotonicNS() uint64   { return 0 }
func TimerFunction() uint64 { return 0 }
