//go:build renvo

package engine

// A Renvo-built host cannot enter the native mapping/session path. Keep a
// violated assumption explicit rather than substituting a non-atomic read.
func nativeStopRequested(stop *uint32) bool { panic("native RFE mapping unavailable") }
