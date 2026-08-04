//go:build !renvo && windows && (amd64 || 386 || arm64)

package runimage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"renvo.dev/internal/linkedimage"
)

const (
	windowsMemCommit       = 0x1000
	windowsMemReserve      = 0x2000
	windowsMemRelease      = 0x8000
	windowsPageRead        = 0x02
	windowsPageReadWrite   = 0x04
	windowsPageExecuteRead = 0x20
)

var (
	windowsKernel32              = syscall.NewLazyDLL("kernel32.dll")
	windowsVirtualAlloc          = windowsKernel32.NewProc("VirtualAlloc")
	windowsVirtualProtect        = windowsKernel32.NewProc("VirtualProtect")
	windowsVirtualFree           = windowsKernel32.NewProc("VirtualFree")
	windowsGetCurrentProcess     = windowsKernel32.NewProc("GetCurrentProcess")
	windowsFlushInstructionCache = windowsKernel32.NewProc("FlushInstructionCache")
)

func runNative(image linkedimage.Image, script string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) Result {
	if stdin != os.Stdin || stdout != os.Stdout || stderr != os.Stderr {
		return Result{ExitCode: 1, Err: errors.New("in-process execution requires the process standard streams")}
	}
	entry, memorySize, preferredBase, wordSize, segments, imports, ok := linkedimage.WindowsLayout(image.Native)
	if !ok || wordSize != int(unsafe.Sizeof(uintptr(0))) {
		return Result{ExitCode: 1, Err: errors.New("invalid Windows linked-image layout")}
	}
	relocations, ok := linkedimage.BaseRelocations(image.Native, memorySize)
	if !ok {
		return Result{ExitCode: 1, Err: errors.New("invalid Windows linked-image relocations")}
	}
	memorySize = pageAlign(memorySize, 4096)
	base, _, callErr := windowsVirtualAlloc.Call(
		0, uintptr(memorySize), windowsMemCommit|windowsMemReserve, windowsPageReadWrite,
	)
	if base == 0 {
		return Result{ExitCode: 1, Err: windowsCallError("VirtualAlloc image", callErr)}
	}
	defer windowsVirtualFree.Call(base, 0, windowsMemRelease)
	memory := unsafe.Slice((*byte)(unsafe.Pointer(base)), memorySize)
	for i := range segments {
		segment := segments[i]
		if segment.FileSize != 0 {
			copy(memory[segment.Address:segment.Address+segment.FileSize], image.Native[segment.FileOffset:segment.FileOffset+segment.FileSize])
		}
	}
	delta := base - uintptr(preferredBase)
	for _, address := range relocations {
		value := binary.LittleEndian.Uint32(memory[address : address+4])
		binary.LittleEndian.PutUint32(memory[address:address+4], value+uint32(delta))
	}
	libraries := make(map[string]*syscall.DLL)
	defer func() {
		for _, library := range libraries {
			_ = library.Release()
		}
	}()
	for _, item := range imports {
		library := libraries[item.Library]
		if library == nil {
			var err error
			library, err = syscall.LoadDLL(item.Library)
			if err != nil {
				return Result{ExitCode: 1, Err: fmt.Errorf("load %s: %w", item.Library, err)}
			}
			libraries[item.Library] = library
		}
		procedure, err := library.FindProc(item.Name)
		if err != nil {
			return Result{ExitCode: 1, Err: fmt.Errorf("resolve %s!%s: %w", item.Library, item.Name, err)}
		}
		if wordSize == 8 {
			binary.LittleEndian.PutUint64(memory[item.Address:item.Address+8], uint64(procedure.Addr()))
		} else {
			binary.LittleEndian.PutUint32(memory[item.Address:item.Address+4], uint32(procedure.Addr()))
		}
	}
	for _, segment := range segments {
		if segment.MemorySize == 0 {
			continue
		}
		protection := uintptr(windowsPageRead)
		if segment.Permissions&2 != 0 {
			protection = windowsPageReadWrite
		} else if segment.Permissions&1 != 0 {
			protection = windowsPageExecuteRead
		}
		start := pageFloor(segment.Address, 4096)
		end := pageAlign(segment.Address+segment.MemorySize, 4096)
		var oldProtection uintptr
		status, _, err := windowsVirtualProtect.Call(
			base+uintptr(start), uintptr(end-start), protection,
			uintptr(unsafe.Pointer(&oldProtection)),
		)
		if status == 0 {
			return Result{ExitCode: 1, Err: windowsCallError("VirtualProtect image", err)}
		}
	}
	process, _, processErr := windowsGetCurrentProcess.Call()
	if process == 0 {
		return Result{ExitCode: 1, Err: windowsCallError("GetCurrentProcess", processErr)}
	}
	status, _, flushErr := windowsFlushInstructionCache.Call(process, base, uintptr(memorySize))
	if status == 0 {
		return Result{ExitCode: 1, Err: windowsCallError("FlushInstructionCache", flushErr)}
	}
	stack, _, stackErr := windowsVirtualAlloc.Call(
		0, jitStackSize, windowsMemCommit|windowsMemReserve, windowsPageReadWrite,
	)
	if stack == 0 {
		return Result{ExitCode: 1, Err: windowsCallError("VirtualAlloc stack", stackErr)}
	}
	defer windowsVirtualFree.Call(stack, 0, windowsMemRelease)
	argStorage, argWords, envStorage, envWords := jitArguments(script, args, env)
	runtime.LockOSThread()
	exitCode := callJIT(
		base+uintptr(entry), stack+jitStackSize,
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

func windowsCallError(operation string, err error) error {
	if errno, ok := err.(syscall.Errno); ok && errno == 0 {
		return errors.New(operation + " failed")
	}
	return fmt.Errorf("%s: %w", operation, err)
}
