// Package runtime defines the target-provided execution hooks used by Renvo's
// goroutine and channel lowering. It is intentionally a policy boundary: the
// compiler preserves Go evaluation and channel semantics while a handler owns
// scheduling, stacks, blocking, wakeups, and channel storage.
package runtime

import "unsafe"

// Entry is an opaque whole-program goroutine entry identifier. It is not
// necessarily a native function address. Invoke it only through Call.
type Entry uintptr

// Context identifies compiler-generated persistent storage for one goroutine
// invocation.
type Context uintptr

// Channel is an implementation-defined channel handle. The zero value is a
// nil channel.
type Channel uintptr

// Status reports failures which the compiler-owned wrapper converts to the
// corresponding recoverable runtime panic.
type Status int

const (
	StatusOK Status = iota
	StatusClosed
	StatusInvalid
)

// SelectDirection identifies the operation represented by a select case.
type SelectDirection int

const (
	SelectReceive SelectDirection = iota
	SelectSend
)

// ChanSelectValue is the type-erased descriptor for one select case. Value is
// the address of a staged send value or receive destination. ReceiveOK is an
// optional address of the receive's boolean result.
type ChanSelectValue struct {
	Channel   Channel
	Value     unsafe.Pointer
	ReceiveOK unsafe.Pointer
	Direction SelectDirection
}

// Internal aliases give compiler-generated linked code stable names without
// exposing backend instructions or depending on a package's chosen import
// name. They are aliases, so handlers still receive []ChanSelectValue.
type renvo_runtime_ChanSelectValue = ChanSelectValue
type renvo_runtime_Channel = Channel
type renvo_runtime_Pointer = unsafe.Pointer

const (
	renvo_runtime_SelectReceive = SelectReceive
	renvo_runtime_SelectSend    = SelectSend
)

// GoHandler supplies target scheduling and channel operations. Version 1
// handlers must serialize execution of Renvo code even when they use separate
// stacks. Value pointers are valid only for the duration of their operation.
type GoHandler interface {
	Spawn(entry uintptr, context uintptr)
	ChanCreate(elementSize uintptr, capacity int) uintptr
	ChanSend(channel uintptr, value unsafe.Pointer) int
	ChanReceive(channel uintptr, value unsafe.Pointer) bool
	ChanSelect(cases []ChanSelectValue, hasDefault bool) (int, int)
	ChanClose(channel uintptr) int
	ChanLen(channel uintptr) int
	ChanCap(channel uintptr) int
}

var activeHandler GoHandler

// EnableGoroutines installs the program's scheduling and channel handler.
// Registration is process-wide and may occur only once.
func EnableGoroutines(handler GoHandler) {
	if handler == nil {
		panic("runtime: nil goroutine handler")
	}
	if activeHandler != nil {
		panic("runtime: goroutine handler already enabled")
	}
	activeHandler = handler
}

// Call invokes an opaque goroutine entry on the current handler-owned stack.
// The compiler lowers the internal call to its deterministic entry dispatcher
// and installs a fresh thread state for the duration of the invocation.
func Call(entry Entry, context Context) {
	renvo_runtime_StackRun(entry, context)
}

func renvo_runtime_StackRun(entry Entry, context Context) {
	// The target ABI reserves eight bytes for each field even on 32-bit
	// targets. Keeping the storage on the task stack ties its lifetime directly
	// to that execution context and avoids a scheduler-side allocation.
	var state [56]byte
	previous := renvo_runtime_ThreadStateSwap(uintptr(unsafe.Pointer(&state)))
	renvo_runtime_Call(entry, context)
	renvo_runtime_ThreadStateSwap(previous)
}

// renvo_runtime_ThreadStateSwap is a compiler-private target hook. Native
// targets may map the logical state to a reserved register; portable targets
// use an equivalent execution-context slot. Host Go needs no explicit state
// because its runtime already isolates panic state per goroutine.
func renvo_runtime_ThreadStateSwap(next uintptr) uintptr { return 0 }

// StackSupported reports whether the current target provides Renvo's small
// cooperative stack-context primitive. It is false under host Go, whose own
// runtime already supplies independent goroutine stacks.
func StackSupported() bool { return renvo_runtime_StackSupported() }

// StackInit prepares a suspended stack which will invoke Call(entry, context),
// mark done, and return to the scheduler stack when the entry finishes.
// stackTop must be the address immediately beyond writable stack storage.
func StackInit(stackTop uintptr, entry Entry, context Context, done uintptr, scheduler uintptr) uintptr {
	return renvo_runtime_StackInit(stackTop, uintptr(entry), uintptr(context), done, scheduler, renvo_runtime_StackRun)
}

// StackSwitch suspends the current stack in save and resumes restore. Only a
// serialized GoHandler may use this low-level primitive.
func StackSwitch(save uintptr, restore uintptr) {
	renvo_runtime_StackSwitch(save, restore, renvo_runtime_StackRun)
}

func renvo_runtime_StackSupported() bool { return false }
func renvo_runtime_StackInit(stackTop uintptr, entry uintptr, context uintptr, done uintptr, scheduler uintptr, run func(Entry, Context)) uintptr {
	_, _, _, _, _, _ = stackTop, entry, context, done, scheduler, run
	return 0
}
func renvo_runtime_StackSwitch(save uintptr, restore uintptr, run func(Entry, Context)) {
	_, _, _ = save, restore, run
}

func requireHandler() GoHandler {
	if activeHandler == nil {
		panic("runtime: goroutines are not enabled")
	}
	return activeHandler
}

func renvo_runtime_Spawn(entry Entry, context Context) {
	requireHandler().Spawn(uintptr(entry), uintptr(context))
}

func renvo_runtime_ChanCreate(elementSize uintptr, capacity int) uintptr {
	if capacity < 0 {
		panic("runtime: negative channel capacity")
	}
	return requireHandler().ChanCreate(elementSize, capacity)
}

func renvo_runtime_ChanSend(channel Channel, value unsafe.Pointer) {
	status := requireHandler().ChanSend(uintptr(channel), value)
	if status == int(StatusClosed) {
		panic("send on closed channel")
	}
	if status != int(StatusOK) {
		panic("runtime: invalid channel send")
	}
}

func renvo_runtime_ChanReceive(channel Channel, value unsafe.Pointer) bool {
	return requireHandler().ChanReceive(uintptr(channel), value)
}

func renvo_runtime_ChanSelect(cases []ChanSelectValue, hasDefault bool) int {
	index, status := requireHandler().ChanSelect(cases, hasDefault)
	if status == int(StatusClosed) {
		panic("send on closed channel")
	}
	if status != int(StatusOK) {
		panic("runtime: invalid channel select")
	}
	return index
}

func renvo_runtime_ChanClose(channel Channel) {
	status := requireHandler().ChanClose(uintptr(channel))
	if status == int(StatusClosed) {
		panic("close of closed channel")
	}
	if status != int(StatusOK) {
		panic("close of nil channel")
	}
}

func renvo_runtime_ChanLen(channel Channel) int {
	if channel == 0 {
		return 0
	}
	return requireHandler().ChanLen(uintptr(channel))
}

func renvo_runtime_ChanCap(channel Channel) int {
	if channel == 0 {
		return 0
	}
	return requireHandler().ChanCap(uintptr(channel))
}

// renvo_runtime_Call is replaced by the whole-program goroutine dispatcher.
// Keeping a real body gives host tooling a deterministic failure before a
// compiled dispatcher exists.
func renvo_runtime_Call(entry Entry, context Context) {
	panic("runtime: invalid goroutine entry")
}
