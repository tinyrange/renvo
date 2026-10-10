//go:build !renvo

package rfenativebridge

// FaultSite is an exact native scalar access PC, its precise recovery PC,
// and its byte width. The arena owns, validates, sorts, and pins these records.
type FaultSite struct{ PC, Recovery, Width uintptr }
