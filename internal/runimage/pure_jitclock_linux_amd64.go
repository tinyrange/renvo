//go:build !renvo && linux && amd64

package runimage

// JITDUMP_FLAGS_ARCH_TIMESTAMP matches perf's Intel PT clock domain.
const jitTimestampFlags = 1

func jitReadTSC() uint64

func jitTimestamp() (uint64, error) { return jitReadTSC(), nil }
