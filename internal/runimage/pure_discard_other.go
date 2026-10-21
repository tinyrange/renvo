//go:build !renvo && ((!linux && !darwin && !windows) || (!amd64 && !arm64))

package runimage

func pureDiscard(base uintptr, size int) error { return nil }
