//go:build renvo && linux

package main

import "renvo.dev/internal/driver"

// renvo_runtime_Syscall is the frontend's explicit generic Linux syscall
// intrinsic. Its body is only a source-level declaration for the frontend.
func renvo_runtime_Syscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return 0
}

func runLinuxSyscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return renvo_runtime_Syscall(number, first, second, third, fourth, fifth, sixth)
}

// renvo_runtime_CallJIT is lowered to an isolated-stack indirect call.
func renvo_runtime_CallJIT(entry int, stackTop int, argsData int, argsLen int, envData int, envLen int) int {
	return 0
}

func runLinuxJIT(entry int, stackTop int, argsData int, argsLen int, envData int, envLen int) int {
	return renvo_runtime_CallJIT(entry, stackTop, argsData, argsLen, envData, envLen)
}

func configureRunPlatform() {
	driver.SetRunLinuxSyscall(runLinuxSyscall)
	driver.SetRunJITCall(runLinuxJIT)
}
