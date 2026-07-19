package main

import "testing"

func TestDarwinDirectoryIntrinsicRejectsLinuxSyscallProbing(t *testing.T) {
	program := func(number string) []byte {
		return []byte("package main\n" +
			"func syscall(num int, fd int, buf []byte, size int) int { return 0 }\n" +
			"func appMain() int { buf := make([]byte, 32); return syscall(" + number + ", 0, buf, len(buf)) }\n")
	}
	if _, ok := RenvoCompileSourceToBytes(program("61"), "darwin/arm64"); ok {
		t.Fatal("Darwin backend accepted the Linux/aarch64 getdents64 selector")
	}
	if _, ok := RenvoCompileSourceToBytes(program("220"), "darwin/arm64"); ok {
		t.Fatal("Darwin backend accepted the Linux/386 getdents64 selector")
	}
	if _, ok := RenvoCompileSourceToBytes(program("217"), "darwin/arm64"); !ok {
		t.Fatal("Darwin backend rejected its directory-read intrinsic selector")
	}
}
