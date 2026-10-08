package linuxuser

import (
	"encoding/binary"
	"io"
)

func errno(value uint64) uint64 { return ^value + 1 }

// Syscall implements only the documented first ABI slice. It never forwards a
// guest syscall number, path, descriptor or address to the host kernel.
func (p *Process) Syscall(number uint64, args [6]uint64, output, diagnostics io.Writer) (value uint64, exited bool, code int) {
	x := &args
	result := errno(38) // ENOSYS
	switch number {
	case 93, 94: // exit, exit_group (single guest thread)
		return 0, true, int(x[0] & 255)
	case 64: // write; bounded, Linux-compatible short transfers
		writer := output
		if x[0] == 2 {
			writer = diagnostics
		} else if x[0] != 1 {
			result = errno(9)
			break
		}
		if x[2] == 0 {
			result = 0
			break
		}
		length := x[2]
		if length > 65536 {
			length = 65536
		}
		data, err := p.Memory.ReadBytes(x[1], length)
		if err != nil {
			result = errno(14)
			break
		}
		if writer == nil {
			result = uint64(len(data))
			break
		}
		n, err := writer.Write(data)
		if n < 0 || n > len(data) {
			result = errno(5)
		} else if n > 0 || err == nil {
			result = uint64(n)
		} else {
			result = errno(5)
		}
	case 113: // clock_gettime: real elapsed time, never guest instruction ticks
		if x[0] > 1 {
			result = errno(22)
			break
		}
		if p.Clock == nil {
			result = errno(38)
			break
		}
		seconds, nanoseconds, ok := p.Clock(x[0])
		if !ok || nanoseconds < 0 || nanoseconds >= 1000000000 {
			result = errno(5)
			break
		}
		data := make([]byte, 16)
		binary.LittleEndian.PutUint64(data, uint64(seconds))
		binary.LittleEndian.PutUint64(data[8:], uint64(nanoseconds))
		if err := p.Memory.WriteBytes(x[1], data); err != nil {
			result = errno(14)
		} else {
			result = 0
		}
	case 172, 178: // getpid, gettid: one virtual process/thread
		result = 1
	case 214: // brk: return old break on failure, as the Linux kernel ABI does
		requested := x[0]
		result = p.Break
		if requested == 0 || requested < p.breakBase || requested-p.breakBase > 16<<20 || !validRange(requested, 0) {
			break
		}
		oldEnd, newEnd := pageUp(p.Break), pageUp(requested)
		var err error
		if newEnd > oldEnd {
			err = p.Memory.Map(oldEnd, newEnd-oldEnd, ReadPermission|WritePermission)
		} else if newEnd < oldEnd {
			err = p.Memory.Unmap(newEnd, oldEnd-newEnd)
		}
		if err == nil {
			p.Break = requested
			result = requested
		}
	case 222: // mmap: anonymous private, no fixed/hint mapping yet
		if x[0] != 0 || x[1] == 0 || x[1] > MemoryLimit || x[2]&^uint64(7) != 0 || x[3] != 0x22 || x[4] != ^uint64(0) || x[5] != 0 {
			result = errno(22)
			break
		}
		length := pageUp(x[1])
		address := p.mmapNext
		if err := p.Memory.Map(address, length, uint8(x[2])); err != nil {
			result = errno(12)
		} else {
			result = address
			p.mmapNext += length + PageSize
		}
	case 215: // munmap
		if x[1] == 0 || x[1] > MemoryLimit || x[0]%PageSize != 0 {
			result = errno(22)
			break
		}
		if err := p.Memory.Unmap(x[0], pageUp(x[1])); err != nil {
			result = errno(22)
		} else {
			result = 0
		}
	case 226: // mprotect
		if x[2]&^uint64(7) != 0 || x[0]%PageSize != 0 || x[1] > MemoryLimit {
			result = errno(22)
			break
		}
		if x[1] == 0 {
			result = 0
			break
		}
		if err := p.Memory.Protect(x[0], pageUp(x[1]), uint8(x[2])); err != nil {
			result = errno(12)
		} else {
			result = 0
		}
	}
	return result, false, 0
}
