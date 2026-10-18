//go:build renvo && browser

package runtime

import (
	"encoding/binary"
	"fmt"
	"os"
	"renvo.dev/internal/arena"
	"syscall"
	"unsafe"
)

// Browser Native installs generated WASM blocks into a bounded worker-owned
// arena. The WASI device is a host-service channel, not guest filesystem I/O;
// installed blocks import the emulator's existing linear memory.
type Native struct {
	layout                                   WasmMemoryLayout
	device                                   *os.File
	capacity                                 int
	gateOwner                                int
	deviceOwner                              int
	gateBytes                                int
	suspended                                bool
	entries                                  []browserEntry
	entryCount                               int
	dispatchEntry, dispatchWords, dispatchPC int
	ownedContexts                            map[*MemoryContext]bool
	codeVersionedContexts                    map[*MemoryContext]bool
	codeVersionedLoops                       bool
	Bytes                                    int
	SymbolErrors                             uint64
	response                                 [4]byte
	callContext                              *MemoryContext
	refillContext                            *MemoryContext
	refill                                   func(*MemoryContext) bool
	refillClock                              *uint64
	refillLeft                               int
	refillRunning                            bool
	refillStack                              []uint64
	proofRequest                             []byte // fixed proof module and synchronous admission header
}

type browserProof struct {
	pc           uint64
	instructions int
}

// Host entries are positive multiples of 16. A zero word count denotes an
// unused slot; validated compiled blocks always have at least one state word.
// The index is bounded by the host table, independent of guest addresses.
type browserEntry struct {
	words, loopInstructions int
	bytes                   int
	memory                  bool
	proof                   browserProof
}

const NativeSessionLimit = 65536

// The host reserves entry zero and holds two table slots (code and proof) for
// each remaining entry. Byte-rich presets can exhaust these slots first.
const browserNativeEntryLimit = 16384 - 1

func (n *Native) EntryBudgetNeedsRecycle() bool {
	return n != nil && n.entryCount >= browserNativeEntryLimit*7/8
}

func browserEntrySlot(entry int) int {
	if entry <= 0 || entry&15 != 0 || entry>>4 > browserNativeEntryLimit {
		return 0
	}
	return entry >> 4
}

func (n *Native) entryMetadata(entry int) *browserEntry {
	slot := browserEntrySlot(entry)
	if n == nil || n.suspended || slot == 0 || slot >= len(n.entries) || n.entries[slot].words == 0 {
		return nil
	}
	return &n.entries[slot]
}

func NativeSessionsAvailable() bool { return true }

func NewNative(capacity int) (*Native, error) { return newBrowserNative(capacity, nil) }

// The replacement has a fresh owner identity and fresh host table. Only empty
// metadata backing arrays are transferred from an already closed owner.
func NewNativeReusing(capacity int, old *Native) (*Native, error) {
	if old == nil || old.device != nil {
		return NewNative(capacity)
	}
	return newBrowserNative(capacity, old)
}
func newBrowserNative(capacity int, old *Native) (*Native, error) {
	if capacity < 4096 || capacity > 64<<20 || unsafe.Sizeof(uintptr(0)) != 4 {
		return nil, fmt.Errorf("invalid browser compiled arena")
	}
	file, err := os.OpenFile("/dev/renvo-rfe", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	n := &Native{device: file, capacity: capacity, layout: wasmMemoryLayout()}
	if old != nil {
		n.refillStack, old.refillStack = old.refillStack, nil
		n.entries, n.ownedContexts = old.entries, old.ownedContexts
		n.codeVersionedContexts = old.codeVersionedContexts
		n.proofRequest, old.proofRequest = old.proofRequest, nil
		old.entries, old.ownedContexts = nil, nil
		old.codeVersionedContexts = nil
	} else {
		n.ownedContexts = map[*MemoryContext]bool{}
		n.codeVersionedContexts = map[*MemoryContext]bool{}
	}
	if n.ownedContexts == nil {
		n.ownedContexts = map[*MemoryContext]bool{}
	}
	if n.codeVersionedContexts == nil {
		n.codeVersionedContexts = map[*MemoryContext]bool{}
	}
	if n.refillStack == nil {
		n.refillStack = make([]uint64, 40960)
	}
	if n.proofRequest == nil {
		// Keep the canonical proof template in each synchronous admission packet
		// for host validation and full original per-entry byte accounting. The
		// arena gate checks write-once private tuples, never packet references.
		code := emitWasmProof()
		n.proofRequest = make([]byte, 24, len(code)+24)
		n.proofRequest = append(n.proofRequest, code...)
		binary.LittleEndian.PutUint32(n.proofRequest, 3)
	}
	gate := emitWasmGate()
	packet := make([]byte, 8, 8+len(gate))
	binary.LittleEndian.PutUint32(packet[4:], uint32(capacity))
	packet = append(packet, gate...)
	if n.gateOwner, err = n.exchange(packet); err != nil {
		file.Close()
		return nil, err
	}
	n.Bytes += len(gate)
	n.gateBytes = len(gate)
	n.deviceOwner = n.gateOwner
	return n, nil
}
func (n *Native) exchange(request []byte) (int, error) {
	if n == nil || n.device == nil {
		return 0, fmt.Errorf("closed browser arena")
	}
	// Requests and reply buffers are caller-owned; successful File I/O changes
	// only existing file-position fields and the preowned response. Its ABI
	// call temporaries must not accumulate once per compiled dispatch.
	low, high := arena.Mark(), arena.PersistMark()
	written, err := n.device.Write(request)
	if err != nil || written != len(request) {
		return 0, fmt.Errorf("browser JIT request failed")
	}
	read, err := n.device.Read(n.response[:])
	if err != nil || read != 4 {
		return 0, fmt.Errorf("browser JIT reply failed")
	}
	result := int(int32(binary.LittleEndian.Uint32(n.response[:])))
	if result < 0 {
		return 0, fmt.Errorf("browser JIT host rejected request %d", result)
	}
	arena.Rewind(low)
	arena.PersistReset(high)
	return result, nil
}
func (n *Native) Compile(ops []Op, words int) (int, error) {
	return n.compileWasm(ops, words, nil)
}

// Installation consumes the module synchronously. Only entry/size integers
// escape the temporary arena scope; maps are updated after its reclamation.
func (n *Native) compileWasm(ops []Op, words int, layout *WasmMemoryLayout) (int, error) {
	return n.compileWasmBody(ops, words, layout, 0)
}
func (n *Native) compileWasmBody(ops []Op, words int, layout *WasmMemoryLayout, instructions int) (int, error) {
	return n.compileWasmMode(ops, words, layout, instructions, false)
}
func (n *Native) compileWasmMode(ops []Op, words int, layout *WasmMemoryLayout, instructions int, codeVersions bool) (int, error) {
	low, high := arena.Mark(), arena.PersistMark()
	var code []byte
	var err error
	if instructions == 0 {
		code, err = emitWasmBlock(ops, words, layout)
	} else {
		code, err = emitWasmLoopMode(ops, words, instructions, layout, codeVersions)
	}
	if err != nil {
		return 0, err // error allocations remain live on failure
	}
	size := len(code)
	entry, err := n.exchangeWasm(code, words, layout != nil)
	if err != nil {
		return 0, err
	}
	arena.Rewind(low)
	arena.PersistReset(high)
	if err := n.recordWasm(entry, words, layout != nil, size); err != nil {
		return 0, err
	}
	return entry, nil
}
func (n *Native) recordWasm(entry, words int, memory bool, size int) error {
	slot := browserEntrySlot(entry)
	if slot == 0 || words < 1 || size < 0 || n.entryMetadata(entry) != nil {
		return fmt.Errorf("invalid browser installed entry")
	}
	if slot >= len(n.entries) {
		length := len(n.entries) * 2
		if length < 16 {
			length = 16
		}
		for length <= slot {
			length *= 2
		}
		entries := make([]browserEntry, length)
		copy(entries, n.entries)
		n.entries = entries
	}
	n.entries[slot] = browserEntry{words: words, memory: memory, bytes: size}
	n.entryCount++
	n.Bytes += size
	return nil
}
func (n *Native) installWasm(code []byte, words int, memory bool) (int, error) {
	entry, err := n.exchangeWasm(code, words, memory)
	if err == nil {
		err = n.recordWasm(entry, words, memory, len(code))
	}
	return entry, err
}
func (n *Native) exchangeWasm(code []byte, words int, memory bool) (int, error) {
	if n.entryCount >= browserNativeEntryLimit {
		return 0, fmt.Errorf("browser compiled entry slots full")
	}
	if len(code) > n.capacity-n.Bytes {
		return 0, fmt.Errorf("browser compiled arena full")
	}
	packet := make([]byte, 16, len(code)+16)
	binary.LittleEndian.PutUint32(packet, 1)
	binary.LittleEndian.PutUint32(packet[4:], uint32(words))
	if memory {
		binary.LittleEndian.PutUint32(packet[8:], 1)
	}
	binary.LittleEndian.PutUint32(packet[12:], uint32(unsafe.Sizeof(MemoryContext{})))
	packet = append(packet, code...)
	return n.exchange(packet)
}
func (n *Native) Call(entry int, state []uint64) error { return n.callWasm(entry, state, nil) }
func (n *Native) callWasm(entry int, state []uint64, context *MemoryContext) error {
	metadata := n.entryMetadata(entry)
	if metadata == nil || metadata.words != len(state) || len(state) == 0 {
		return fmt.Errorf("invalid browser block state")
	}
	if metadata.memory != (context != nil) {
		return fmt.Errorf("browser compiled context mismatch")
	}
	if n.device == nil {
		return fmt.Errorf("closed browser arena")
	}
	// Synchronous Wasm invocation finishes every checked effect before return.
	low, high := arena.Mark(), arena.PersistMark()
	n.prepareFastRefill(context)
	n.callContext = context
	// The reserved Wasm syscall enters a host-private checked arena gate.
	// No fd_write, JavaScript call, shared public descriptor, or copied
	// context confers the gate's immutable owner/dimension authority.
	status := syscall.BrowserInvoke(n.gateOwner, entry, int(uintptr(unsafe.Pointer(&state[0]))), len(state), int(uintptr(unsafe.Pointer(context))))
	n.callContext = nil
	if context != nil {
		context.wasmServiceSP, context.wasmServiceFP = 0, 0
	}
	if status != 0 {
		return fmt.Errorf("browser compiled dispatch failed")
	}
	arena.Rewind(low)
	arena.PersistReset(high)
	return nil
}
func (n *Native) CallBatch(state []uint64, limit int, next func(int) (int, bool, error)) (int, error) {
	if limit < 1 || limit > 64 || next == nil {
		return 0, fmt.Errorf("invalid browser batch")
	}
	for completed := 0; completed <= limit; completed++ {
		entry, ok, err := next(completed)
		if err != nil || !ok || completed == limit {
			return completed, err
		}
		if err = n.Call(entry, state); err != nil {
			return completed, err
		}
	}
	return 0, fmt.Errorf("invalid browser batch progress")
}
func (n *Native) Close() error {
	if n == nil {
		return nil
	}
	if n.device != nil && n.deviceOwner != 0 {
		var packet [8]byte
		binary.LittleEndian.PutUint32(packet[:], 6)
		binary.LittleEndian.PutUint32(packet[4:], uint32(n.deviceOwner))
		allowed, err := n.exchange(packet[:])
		if err != nil {
			return err
		}
		if allowed != 1 {
			// A stale copy must not close the transferred file or clear arrays
			// now owned by its replacement. Only forget this copy's handles.
			n.device, n.gateOwner, n.deviceOwner = nil, 0, 0
			n.entries, n.ownedContexts, n.codeVersionedContexts = nil, nil, nil
			n.refillStack, n.proofRequest = nil, nil
			n.entryCount, n.dispatchEntry, n.dispatchWords = 0, 0, 0
			n.codeVersionedLoops = false
			return nil
		}
	}
	n.EndFastRefill()
	for context := range n.ownedContexts {
		if context.linkOwner == n {
			context.ClearLinks()
			context.linkOwner = nil
		}
	}
	clear(n.entries)
	n.entryCount = 0
	n.dispatchEntry, n.dispatchWords, n.dispatchPC = 0, 0, 0
	clear(n.ownedContexts)
	clear(n.codeVersionedContexts)
	n.codeVersionedLoops = false
	file := n.device
	n.device = nil
	n.gateOwner, n.deviceOwner = 0, 0
	if file == nil {
		return nil
	}
	return file.Close()
}
func wasmMemoryLayout() WasmMemoryLayout {
	var m MemoryContext
	var p NativePage
	var d NativeLink
	return WasmMemoryLayout{PageSlots: NativePageSlots, ServiceWrite: uint32(unsafe.Offsetof(m.RefillAccess)), ServiceSP: uint32(unsafe.Offsetof(m.wasmServiceSP)), ServiceFP: uint32(unsafe.Offsetof(m.wasmServiceFP)),
		Retired: uint32(unsafe.Offsetof(m.Retired)), Status: uint32(unsafe.Offsetof(m.Status)), Address: uint32(unsafe.Offsetof(m.Address)), Clock: uint32(unsafe.Offsetof(m.Clock)), Pages: uint32(unsafe.Offsetof(m.browserPages)),
		Total: uint32(unsafe.Offsetof(m.Total)), Remaining: uint32(unsafe.Offsetof(m.Remaining)), MemoryTotal: uint32(unsafe.Offsetof(m.MemoryTotal)), Blocks: uint32(unsafe.Offsetof(m.browserLinks)), BlockSets: nativeLinkSets, BlocksIndirect: true, LoopExits: uint32(unsafe.Offsetof(m.LoopExits)), LoopIterations: uint32(unsafe.Offsetof(m.LoopIterations)), MemoryRetired: uint32(unsafe.Offsetof(m.wasmMemoryRetired)), Linked: uint32(unsafe.Offsetof(m.wasmLinked)), LinkSource: uint32(unsafe.Offsetof(m.wasmLinkSource)),
		DescriptorSize: uint32(unsafe.Sizeof(d)), DescriptorPC: uint32(unsafe.Offsetof(d.PC)), DescriptorEntry: uint32(unsafe.Offsetof(d.Entry)), DescriptorInstructions: uint32(unsafe.Offsetof(d.Instructions)), DescriptorHot: uint32(unsafe.Offsetof(d.Reserved)),
		PageSize: uint32(unsafe.Sizeof(p)), PageNumber: uint32(unsafe.Offsetof(p.Number)), PageData: uint32(unsafe.Offsetof(p.Data)), PagePermissions: uint32(unsafe.Offsetof(p.Permissions)), PageEpoch: uint32(unsafe.Offsetof(p.Epoch)),
	}
}
func (n *Native) CompileMemory(ops []Op, words int) (int, error) {
	return n.compileWasm(ops, words, &n.layout)
}
func (n *Native) CallMemory(entry int, state []uint64, memory *MemoryContext) error {
	metadata := n.entryMetadata(entry)
	if memory == nil || metadata == nil || metadata.loopInstructions != 0 || entry == n.dispatchEntry {
		return fmt.Errorf("invalid browser memory entry")
	}
	return n.callWasm(entry, state, memory)
}
func (n *Native) CallBatchMemory(state []uint64, memory *MemoryContext, limit int, next func(int) (int, bool, error)) (int, error) {
	if memory == nil || limit < 1 || limit > 64 || next == nil {
		return 0, fmt.Errorf("invalid browser memory batch")
	}
	for completed := 0; completed <= limit; completed++ {
		entry, ok, err := next(completed)
		if err != nil || !ok || completed == limit {
			return completed, err
		}
		metadata := n.entryMetadata(entry)
		if metadata != nil && metadata.memory {
			err = n.CallMemory(entry, state, memory)
		} else {
			err = n.Call(entry, state)
		}
		if err != nil {
			return completed, err
		}
	}
	return 0, fmt.Errorf("invalid browser memory progress")
}
func (n *Native) EnablePerfMap(monotonic ...bool) error  { return nil }
func (n *Native) NameEntry(entry int, name string) error { return nil }
