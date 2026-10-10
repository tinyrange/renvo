//go:build !renvo

package runimage

import (
	"fmt"
	"io"
	"renvo.dev/internal/rfenativebridge"
	"runtime"
	"sync"
	"unsafe"
)

// CodeArena owns bounded native code storage for short state transformations
// and internal bounded dispatchers whose calls target checked leaf entries.
// Installation, calls, and Close are serialized. Entries are offsets, so callers
// cannot accidentally invoke an arbitrary host address.
type CodeArena struct {
	faults       []rfenativebridge.FaultSite
	mu           sync.Mutex
	base         uintptr
	size, used   int
	symbols      io.Writer
	jitSymbols   JITCodeWriter
	extents      []codeExtent // sorted exact extents, one per installed image
	entries      []arenaEntry // one private admission record per 16-byte code offset
	stack        []byte
	broken       bool
	linkedView   [4]uint64 // refreshed under mu after every successful installation
	targetVictim uint64
	targets      [1024]linkedTarget         // opaque prepared proofs; no executable pointers
	sessionEpoch uint64                     // arena-serialized generation, never reused without clearing proofs
	sessionImage *[sessionImageWords]uint64 // dedicated pointer-free foreign ABI storage
	sessionState *[256]uint64               // independent of enclosing Go owner allocations
}

type codeExtent struct{ offset, length uint32 }

// arenaEntry is read only during a serialized native call. Linked is an
// immutable admission key, never a caller-controlled host pointer.
type arenaEntry struct {
	Words         uint16
	Linked        uint16
	Body          uint32 // private relative body offset; zero enters the installed start
	Instructions  uint32 // exact immutable loop length, independent of family bits
	AdmittedCount uint32 // exact cold-admitted descriptor count
}

func NewCodeArena(size int) (*CodeArena, error) {
	if unsafe.Sizeof(linkedTarget{}) != 64 || unsafe.Offsetof(linkedTarget{}.CheckedSession) != 32 || unsafe.Offsetof(linkedTarget{}.DescriptorOffset) != 40 || unsafe.Offsetof(linkedTarget{}.BodyFlags) != 24 || unsafe.Sizeof(arenaEntry{}) != 16 || unsafe.Offsetof(arenaEntry{}.Linked) != 2 || unsafe.Offsetof(arenaEntry{}.Body) != 4 || unsafe.Offsetof(arenaEntry{}.Instructions) != 8 || unsafe.Offsetof(arenaEntry{}.AdmittedCount) != 12 {
		return nil, fmt.Errorf("unsupported native admission ABI")
	}
	if size < 4096 || size > 64<<20 {
		return nil, fmt.Errorf("invalid code arena size")
	}
	size = (size + 16383) &^ 16383
	base, err := pureMap(size)
	if err != nil {
		return nil, err
	}
	return &CodeArena{base: base, size: size, stack: make([]byte, 64<<10)}, nil
}

// Install accepts a legacy block with no declared state size.
func (a *CodeArena) Install(code []byte) (int, error) {
	return a.install(code, 65535)
}

// InstallBlock installs a block whose exact state size is checked on every call,
// under the same lock that protects installation, execution and Close.
func (a *CodeArena) InstallBlock(code []byte, words int) (int, error) {
	if words < 1 || words > 256 {
		return 0, fmt.Errorf("invalid native block state size")
	}
	return a.install(code, uint16(words))
}

// InstallLoopBlock admits a bounded loop only as a native dispatcher target.
// Exact body length has its own immutable field; family bits no longer pack it.
func (a *CodeArena) InstallLoopBlock(code []byte, words, instructions int) (int, error) {
	if words < 1 || words > 256 || instructions < 1 || instructions > 256 {
		return 0, fmt.Errorf("invalid native loop dimensions")
	}
	return a.installLoop(code, uint16(words)|32768|8192, instructions)
}

// InstallContextBlock requires a non-nil trusted host context on every call.
func (a *CodeArena) InstallContextBlock(code []byte, words int) (int, error) {
	if words < 1 || words > 256 {
		return 0, fmt.Errorf("invalid native block state size")
	}
	return a.install(code, uint16(words)|32768)
}

func (a *CodeArena) installLoop(code []byte, words uint16, instructions int) (int, error) {
	return a.installRecord(code, words, 0, instructions)
}

func (a *CodeArena) install(code []byte, words uint16) (int, error) {
	return a.installWithBody(code, words, 0)
}

func (a *CodeArena) installWithBody(code []byte, words uint16, body int) (int, error) {
	return a.installRecord(code, words, body, 0)
}

func (a *CodeArena) installRecord(code []byte, words uint16, body, instructions int) (int, error) {
	return a.installFaultRecord(code, words, body, instructions, nil)
}

// InstallFaultLoopBlock registers only immutable scalar access/recovery sites.
// Offsets cannot escape the installed fragment; table growth shares the same
// lock as code installation, native calls, and Close.
func (a *CodeArena) InstallFaultLoopBlock(code []byte, words, instructions int, faults []int) (int, error) {
	if words < 1 || words > 256 || instructions < 1 || instructions > 256 || len(faults)%3 != 0 || !rfenativebridge.EnableFaults() {
		return 0, fmt.Errorf("invalid direct loop dimensions or unavailable fault recovery")
	}
	return a.installFaultRecord(code, uint16(words)|32768|8192, 0, instructions, faults)
}
func (a *CodeArena) installFaultRecord(code []byte, words uint16, body, instructions int, faults []int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	at := (a.used + 15) &^ 15
	if a.base == 0 || a.broken || len(code) == 0 || len(code) > a.size-at || body < 0 || body >= len(code) {
		return 0, fmt.Errorf("native code arena full or closed")
	}
	for i := 0; i < len(faults); i += 3 {
		pc, recovery, width := faults[i], faults[i+1], faults[i+2]
		if pc < 0 || pc >= len(code) || recovery < 0 || recovery >= len(code) || i != 0 && pc <= faults[i-3] || width != 1 && width != 2 && width != 4 && width != 8 {
			return 0, fmt.Errorf("invalid native fault site")
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := pureWritable(a.base, a.size, true); err != nil {
		return 0, err
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(a.base+uintptr(at))), len(code)), code)
	if err := pureSeal(a.base, a.size, at, len(code)); err != nil {
		a.broken = true
		return 0, err
	}
	for i := 0; i < len(faults); i += 3 {
		a.faults = append(a.faults, rfenativebridge.FaultSite{PC: a.base + uintptr(at+faults[i]), Recovery: a.base + uintptr(at+faults[i+1]), Width: uintptr(faults[i+2])})
	}
	a.used = at + len(code)
	slot := at >> 4
	if slot >= len(a.entries) {
		a.entries = append(a.entries, make([]arenaEntry, slot+1-len(a.entries))...)
	}
	a.entries[slot] = arenaEntry{Words: words, Body: uint32(body), Instructions: uint32(instructions)}
	a.extents = append(a.extents, codeExtent{uint32(at), uint32(len(code))})
	a.linkedView = [4]uint64{uint64(a.base), uint64(a.used), uint64(uintptr(unsafe.Pointer(&a.entries[0]))), uint64(len(a.entries))}
	return at, nil
}
func (a *CodeArena) Call(entry int, state []uint64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.callLocked(entry, state, nil)
}

// CallContext keeps context and its host-owned pointer graph alive during a
// checked native call. The installed entry's context requirement is enforced.
func (a *CodeArena) CallContext(entry int, state []uint64, context unsafe.Pointer) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.callLocked(entry, state, context)
}
func (a *CodeArena) callLocked(entry int, state []uint64, context unsafe.Pointer) error {
	if a.base == 0 || a.broken || entry < 0 || entry&15 != 0 || entry>>4 >= len(a.entries) || len(state) == 0 {
		return fmt.Errorf("invalid native block entry")
	}
	words := a.entries[entry>>4].Words
	if words == 0 || words != 65535 && (words&8192 != 0 || int(words&16383) != len(state) || words&32768 != 0 && context == nil) {
		return fmt.Errorf("invalid native block entry")
	}
	top := (uintptr(unsafe.Pointer(&a.stack[len(a.stack)-1])) + 1) &^ 15
	if context == nil {
		callPure(a.base+uintptr(entry), uintptr(unsafe.Pointer(&state[0])), top)
	} else {
		callContext(a.base+uintptr(entry), uintptr(unsafe.Pointer(&state[0])), uintptr(context), top)
	}
	runtime.KeepAlive(context)
	runtime.KeepAlive(state)
	runtime.KeepAlive(a.stack)
	return nil
}
func (a *CodeArena) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 {
		return nil
	}
	err := pureUnmap(a.base, a.size)
	if err == nil {
		a.base = 0
		a.entries = nil
		a.faults = nil
		a.extents = nil
		a.linkedView = [4]uint64{}
		a.symbols = nil
		a.jitSymbols = nil
		a.stack = nil
		a.sessionImage, a.sessionState = nil, nil
	}
	return err
}
