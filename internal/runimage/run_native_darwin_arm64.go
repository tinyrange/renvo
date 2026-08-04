//go:build !renvo && darwin && arm64

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
	darwinProtRead   = 1
	darwinProtWrite  = 2
	darwinProtExec   = 4
	darwinMapPrivate = 2
	darwinMapFixed   = 0x10
	darwinMapJIT     = 0x800
	darwinMapAnon    = 0x1000
	darwinRTLDNow    = 2
)

func runNative(image linkedimage.Image, script string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) Result {
	if stdin != os.Stdin || stdout != os.Stdout || stderr != os.Stderr {
		return Result{ExitCode: 1, Err: errors.New("in-process execution requires the process standard streams")}
	}
	entry, memorySize, segments, imports, libraries, ok := linkedimage.DarwinLayout(image.Native)
	if !ok {
		return Result{ExitCode: 1, Err: errors.New("invalid Darwin linked-image layout")}
	}
	memorySize = pageAlign(memorySize, 16384)
	base, err := darwinMmap(0, uintptr(memorySize), 0, darwinMapPrivate|darwinMapAnon, ^uintptr(0), 0)
	if err != nil {
		return Result{ExitCode: 1, Err: fmt.Errorf("reserve Darwin linked image: %w", err)}
	}
	if base == 0 {
		return Result{ExitCode: 1, Err: errors.New("reserve Darwin linked image returned a null address")}
	}
	defer darwinMunmap(base, uintptr(memorySize))

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	jitWritable := false
	defer func() {
		if jitWritable {
			darwinJITWriteProtect(true)
		}
	}()
	for _, segment := range segments {
		if segment.MemorySize == 0 {
			continue
		}
		start := pageFloor(segment.Address, 16384)
		end := pageAlign(segment.Address+segment.MemorySize, 16384)
		protection := darwinProtRead | darwinProtWrite
		flags := darwinMapPrivate | darwinMapAnon | darwinMapFixed
		if segment.Permissions&1 != 0 {
			protection |= darwinProtExec
			if err = darwinMunmap(base+uintptr(start), uintptr(end-start)); err != nil {
				return Result{ExitCode: 1, Err: fmt.Errorf("release Darwin code reservation: %w", err)}
			}
			flags = darwinMapPrivate | darwinMapAnon | darwinMapJIT
			if !jitWritable {
				darwinJITWriteProtect(false)
				jitWritable = true
			}
		}
		mapped, mapErr := darwinMmap(base+uintptr(start), uintptr(end-start), protection, flags, ^uintptr(0), 0)
		if mapErr != nil || mapped != base+uintptr(start) {
			if mapErr == nil && mapped != 0 {
				_ = darwinMunmap(mapped, uintptr(end-start))
			}
			if mapErr != nil {
				return Result{ExitCode: 1, Err: fmt.Errorf("map Darwin linked-image segment: %w", mapErr)}
			}
			return Result{ExitCode: 1, Err: errors.New("Darwin linked-image segment mapped at the wrong address")}
		}
	}
	for _, segment := range segments {
		if segment.FileSize == 0 {
			continue
		}
		destination := unsafe.Slice((*byte)(unsafe.Pointer(base+uintptr(segment.Address))), segment.FileSize)
		copy(destination, image.Native[segment.FileOffset:segment.FileOffset+segment.FileSize])
	}
	handles := make([]uintptr, len(libraries))
	defer func() {
		for i := len(handles) - 1; i >= 0; i-- {
			if handles[i] != 0 {
				darwinDlclose(handles[i])
			}
		}
	}()
	for i, library := range libraries {
		name := append([]byte(library), 0)
		handles[i] = darwinDlopen(uintptr(unsafe.Pointer(&name[0])), darwinRTLDNow)
		if handles[i] == 0 {
			return Result{ExitCode: 1, Err: fmt.Errorf("load Darwin library %s", library)}
		}
	}
	for _, item := range imports {
		handle := uintptr(0)
		for i, library := range libraries {
			if library == item.Library {
				handle = handles[i]
				break
			}
		}
		if handle == 0 {
			return Result{ExitCode: 1, Err: fmt.Errorf("resolve Darwin library %s", item.Library)}
		}
		name := item.Name
		if len(name) != 0 && name[0] == '_' {
			name = name[1:]
		}
		symbol := append([]byte(name), 0)
		address := darwinDlsym(handle, uintptr(unsafe.Pointer(&symbol[0])))
		if address == 0 {
			return Result{ExitCode: 1, Err: fmt.Errorf("resolve Darwin symbol %s", item.Name)}
		}
		slot := unsafe.Slice((*byte)(unsafe.Pointer(base+uintptr(item.Address))), 8)
		binary.LittleEndian.PutUint64(slot, uint64(address))
	}
	for _, segment := range segments {
		if segment.MemorySize == 0 || segment.Permissions&1 != 0 {
			continue
		}
		protection := 0
		if segment.Permissions&4 != 0 {
			protection |= darwinProtRead
		}
		if segment.Permissions&2 != 0 {
			protection |= darwinProtWrite
		}
		start := pageFloor(segment.Address, 16384)
		end := pageAlign(segment.Address+segment.MemorySize, 16384)
		if err = darwinMprotect(base+uintptr(start), uintptr(end-start), protection); err != nil {
			return Result{ExitCode: 1, Err: fmt.Errorf("protect Darwin linked-image segment: %w", err)}
		}
	}
	darwinInvalidateInstructionCache(base, uintptr(memorySize))
	if jitWritable {
		darwinJITWriteProtect(true)
		jitWritable = false
	}
	stack, err := darwinMmap(
		0, jitStackSize, darwinProtRead|darwinProtWrite,
		darwinMapPrivate|darwinMapAnon, ^uintptr(0), 0,
	)
	if err != nil {
		return Result{ExitCode: 1, Err: fmt.Errorf("map Darwin JIT stack: %w", err)}
	}
	if stack == 0 {
		return Result{ExitCode: 1, Err: errors.New("map Darwin JIT stack returned a null address")}
	}
	defer darwinMunmap(stack, jitStackSize)
	argStorage, argWords, envStorage, envWords := jitArguments(script, args, env)
	exitCode := callJIT(
		base+uintptr(entry), stack+jitStackSize,
		jitWordPointer(argWords), uintptr(len(args)+1),
		jitWordPointer(envWords), uintptr(len(env)),
	)
	runtime.KeepAlive(argStorage)
	runtime.KeepAlive(argWords)
	runtime.KeepAlive(envStorage)
	runtime.KeepAlive(envWords)
	return Result{ExitCode: exitCode, Loader: "jit"}
}

func darwinMmap(address, length uintptr, protection, flags int, fd, offset uintptr) (uintptr, error) {
	result, _, errno := darwinSyscall6(
		darwinMmapAddr, address, length, uintptr(protection), uintptr(flags), fd, offset,
	)
	if errno != 0 {
		return 0, errno
	}
	return result, nil
}

func darwinMprotect(address, length uintptr, protection int) error {
	_, _, errno := darwinSyscall(darwinMprotectAddr, address, length, uintptr(protection))
	if errno != 0 {
		return errno
	}
	return nil
}

func darwinMunmap(address, length uintptr) error {
	_, _, errno := darwinSyscall(darwinMunmapAddr, address, length, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func darwinDlopen(path uintptr, mode int) uintptr {
	result, _, _ := darwinSyscall(darwinDlopenAddr, path, uintptr(mode), 0)
	return result
}

func darwinDlsym(handle, name uintptr) uintptr {
	result, _, _ := darwinSyscall(darwinDlsymAddr, handle, name, 0)
	return result
}

func darwinDlclose(handle uintptr) {
	_, _, _ = darwinSyscall(darwinDlcloseAddr, handle, 0, 0)
}

func darwinInvalidateInstructionCache(address, length uintptr) {
	_, _, _ = darwinSyscall(darwinInvalidateInstructionCacheAddr, address, length, 0)
}

func darwinJITWriteProtect(enabled bool) {
	value := uintptr(0)
	if enabled {
		value = 1
	}
	_, _, _ = darwinSyscall(darwinJITWriteProtectAddr, value, 0, 0)
}

//go:linkname darwinSyscall syscall.syscall
func darwinSyscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname darwinSyscall6 syscall.syscall6
func darwinSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

var (
	darwinMmapAddr                       uintptr
	darwinMprotectAddr                   uintptr
	darwinMunmapAddr                     uintptr
	darwinDlopenAddr                     uintptr
	darwinDlsymAddr                      uintptr
	darwinDlcloseAddr                    uintptr
	darwinInvalidateInstructionCacheAddr uintptr
	darwinJITWriteProtectAddr            uintptr
)

//go:cgo_import_dynamic libc_mmap mmap "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_mprotect mprotect "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_munmap munmap "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_dlopen dlopen "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_dlsym dlsym "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_dlclose dlclose "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_sys_icache_invalidate sys_icache_invalidate "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic libc_pthread_jit_write_protect_np pthread_jit_write_protect_np "/usr/lib/libSystem.B.dylib"
