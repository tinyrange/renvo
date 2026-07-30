package main

import "testing"

func TestHostKernelBTFModuleLayoutWhenAvailable(t *testing.T) {
	data := renvoKernelReadFile("/sys/kernel/btf/vmlinux")
	if len(data) == 0 {
		t.Skip("host kernel BTF is unavailable")
	}
	size, name, init, exit, ok := renvoKernelBTFModuleLayout(data)
	if !ok {
		t.Fatal("host kernel BTF does not expose a usable module layout")
	}
	t.Logf("module layout: size=%d name=%d init=%d exit=%d", size, name, init, exit)
}
