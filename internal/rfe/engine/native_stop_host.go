//go:build !renvo

package engine

import "sync/atomic"

func nativeStopRequested(stop *uint32) bool { return atomic.LoadUint32(stop) != 0 }
