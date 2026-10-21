//go:build !renvo

package engine

import "time"

const paybackClockAvailable = true

type paybackTime = time.Time

func paybackNow() paybackTime                 { return time.Now() }
func paybackElapsed(start paybackTime) uint64 { return uint64(time.Since(start)) }
