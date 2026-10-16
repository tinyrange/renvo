//go:build !renvo && linux && amd64 && !faketime

package systemclock

import (
	"renvo.dev/internal/rfenativebridge"
	_ "unsafe"
)

// The absolute CLOCK_MONOTONIC domain is required to share a deadline with C;
// a relative time.Since origin is not an absolute native deadline. Go retains
// this linkname signature for compatibility. Keep it behind this host adapter.
//
//go:linkname hostMonotonicNS runtime.nanotime
func hostMonotonicNS() int64

func nativeNow() uint64 {
	if !rfenativebridge.Available {
		return 0
	}
	return uint64(hostMonotonicNS())
}
