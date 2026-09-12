//go:build !renvo && darwin && arm64

package runimage

func pureMap(size int) (uintptr, error) {
	return darwinMmap(0, uintptr(size), 7, darwinMapPrivate|darwinMapAnon|darwinMapJIT, ^uintptr(0), 0)
}
func pureWritable(base uintptr, size int, write bool) error {
	darwinJITWriteProtect(!write)
	return nil
}
func pureFlush(base uintptr, size int)       { darwinInvalidateInstructionCache(base, uintptr(size)) }
func pureUnmap(base uintptr, size int) error { return darwinMunmap(base, uintptr(size)) }
