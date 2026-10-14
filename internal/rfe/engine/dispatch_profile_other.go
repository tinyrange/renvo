//go:build renvo || !linux

package engine

func profileMonotonicNS() uint64 { return 0 }
