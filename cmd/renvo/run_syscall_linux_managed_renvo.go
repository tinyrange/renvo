//go:build renvo && linux && renvo_managed

package main

import "renvo.dev/internal/driver"

//renvo:intrinsic renvo_runtime_Syscall
func renvo_runtime_Syscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return 0
}

func runLinuxSyscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return renvo_runtime_Syscall(number, first, second, third, fourth, fifth, sixth)
}

func configureRunPlatform() {
	driver.SetRunLinuxSyscall(runLinuxSyscall)
	// The compiler uses its embedded backend on this registered native stack.
	// Script JIT execution remains disabled until foreign roots/stacks are registered.
}
