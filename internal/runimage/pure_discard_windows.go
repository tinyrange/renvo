//go:build !renvo && windows && (amd64 || arm64)

package runimage

func pureDiscard(base uintptr, size int) error {
	p, _, err := windowsVirtualAlloc.Call(base, uintptr(size), 0x80000, windowsPageReadWrite)
	if p == 0 {
		return err
	}
	return nil
}
