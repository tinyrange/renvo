//go:build !renvo && darwin && arm64

package runimage

func pureMap(size int) (uintptr, error) {
	return darwinMmap(0, uintptr(size), 7, darwinMapPrivate|darwinMapAnon|darwinMapJIT, ^uintptr(0), 0)
}
func pureWritable(base uintptr, size int, write bool) error {
	darwinJITWriteProtect(!write)
	return nil
}
func pureSeal(base uintptr, size, at, count int) error {
	darwinJITWriteProtect(true)
	darwinInvalidateInstructionCache(base+uintptr(at), uintptr(count))
	return nil
}
func pureUnmap(base uintptr, size int) error { return darwinMunmap(base, uintptr(size)) }
