//go:build !renvo && linux

package runimage

import (
	"errors"
	"io"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"renvo.dev/internal/linkedimage"
)

func runNative(image linkedimage.Image, script string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) Result {
	if stdin != os.Stdin || stdout != os.Stdout || stderr != os.Stderr {
		return Result{ExitCode: 1, Err: errors.New("in-process execution requires the process standard streams")}
	}
	entry, memorySize, segments, ok := linkedimage.LinuxLayout(image.Native)
	if !ok {
		return Result{ExitCode: 1, Err: errors.New("invalid Linux linked-image layout")}
	}
	memorySize = pageAlign(memorySize, 4096)
	memory, err := syscall.Mmap(-1, 0, memorySize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return Result{ExitCode: 1, Err: err}
	}
	defer func() { _ = syscall.Munmap(memory) }()
	for i := range segments {
		segment := segments[i]
		if segment.FileSize == 0 {
			continue
		}
		copy(memory[segment.Address:segment.Address+segment.FileSize], image.Native[segment.FileOffset:segment.FileOffset+segment.FileSize])
	}
	for i := range segments {
		segment := segments[i]
		if segment.MemorySize == 0 {
			continue
		}
		protection := 0
		if segment.Permissions&4 != 0 {
			protection |= syscall.PROT_READ
		}
		if segment.Permissions&2 != 0 {
			protection |= syscall.PROT_WRITE
		}
		if segment.Permissions&1 != 0 {
			protection |= syscall.PROT_EXEC
		}
		start := pageFloor(segment.Address, 4096)
		end := pageAlign(segment.Address+segment.MemorySize, 4096)
		if err = syscall.Mprotect(memory[start:end], protection); err != nil {
			return Result{ExitCode: 1, Err: err}
		}
	}
	stack, err := syscall.Mmap(-1, 0, jitStackSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return Result{ExitCode: 1, Err: err}
	}
	defer func() { _ = syscall.Munmap(stack) }()
	argStorage, argWords, envStorage, envWords := jitArguments(script, args, env)
	base := uintptr(unsafe.Pointer(&memory[0]))
	stackTop := uintptr(unsafe.Pointer(&stack[0])) + uintptr(len(stack))
	runtime.LockOSThread()
	exitCode := callJIT(
		base+uintptr(entry), stackTop,
		jitWordPointer(argWords), uintptr(len(args)+1),
		jitWordPointer(envWords), uintptr(len(env)),
	)
	runtime.UnlockOSThread()
	runtime.KeepAlive(argStorage)
	runtime.KeepAlive(argWords)
	runtime.KeepAlive(envStorage)
	runtime.KeepAlive(envWords)
	return Result{ExitCode: exitCode, Loader: "jit"}
}
