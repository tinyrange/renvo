//go:build renvo

package engine

// Renvo-built hosts support the portable interpreter and IR tiers, but have no
// host monotonic clock for measured adaptive promotion. New rejects Adaptive
// before any clock access rather than fabricating timings or importing time.
const paybackClockAvailable = false

type paybackTime struct{}

func paybackNow() paybackTime                 { panic("adaptive RFE timing unavailable") }
func paybackElapsed(start paybackTime) uint64 { panic("adaptive RFE timing unavailable") }
