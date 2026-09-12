//go:build !renvo && ((!linux && !windows && !darwin) || (darwin && !arm64) || (!amd64 && !arm64))

package runimage

import "fmt"

func pureMap(size int) (uintptr, error) {
	return 0, fmt.Errorf("native RFE blocks are unsupported on this host")
}
func pureWritable(base uintptr, size int, write bool) error {
	return fmt.Errorf("unsupported native RFE host")
}
func pureSeal(base uintptr, size, at, count int) error {
	return fmt.Errorf("unsupported native RFE host")
}
func pureUnmap(base uintptr, size int) error { return nil }
