// Package serial provides Renvo's small single-threaded reference goroutine
// and channel handler. It schedules one Renvo entry at a time and uses blocked
// calls as cooperative suspension points. It is deliberately not a parallel
// runtime or an RTOS.
package serial

import (
	"unsafe"

	runtime "renvo.dev/x/runtime"
)

const stackSize = 64 * 1024
const selectPendingStatus = -1

type job struct {
	entry   uintptr
	context uintptr
	stack   *stackStorage
	sp      uintptr
	done    bool
	queued  bool
}

type stackStorage struct {
	data [65536]byte
}

type waiter struct {
	value  []byte
	target uintptr
	done   bool
	open   bool
	status int
	task   *job
}

type selectWait struct {
	bindings []*selectBinding
	selected int
	status   int
}

type selectBinding struct {
	wait  *waiter
	group *selectWait
	index int
	owner *channel
	send  bool
}

type channel struct {
	elementSize int
	capacity    int
	closed      bool
	buffer      []byte
	head        int
	length      int
	senders     []*waiter
	receivers   []*waiter
	selects     []*selectBinding
}

// Handler is the serialized reference implementation of runtime.GoHandler.
// Its zero value is ready for use.
type Handler struct {
	jobs       []*job
	jobHead    int
	channels   []*channel
	selectNext int
	scheduler  uintptr
	current    *job
}

// New returns an empty serialized handler.
func New() *Handler { return new(Handler) }

// Spawn queues a compiler-generated entry. A blocked channel operation runs
// queued entries one at a time through runtime.Call.
func (h *Handler) Spawn(entry uintptr, context uintptr) {
	h.enqueue(&job{entry: entry, context: context})
}

// Drain runs queued entries until the queue is empty. Programs usually
// synchronize through channels instead; Drain is useful for tests and event
// loops that want all currently runnable work to complete.
func (h *Handler) Drain() {
	for h.runOne() {
	}
}

func (h *Handler) runOne() bool {
	for h.jobHead < len(h.jobs) && h.jobs[h.jobHead].done {
		h.jobHead++
	}
	if h.jobHead >= len(h.jobs) {
		h.compactJobs()
		return false
	}
	item := h.jobs[h.jobHead]
	h.jobHead++
	item.queued = false
	if runtime.StackSupported() {
		if item.sp == 0 {
			item.stack = new(stackStorage)
			top := uintptr(unsafe.Pointer(&item.stack.data[0])) + uintptr(stackSize)
			item.sp = runtime.StackInit(top, runtime.Entry(item.entry), runtime.Context(item.context), uintptr(unsafe.Pointer(&item.done)), uintptr(unsafe.Pointer(&h.scheduler)))
		}
		h.current = item
		runtime.StackSwitch(uintptr(unsafe.Pointer(&h.scheduler)), item.sp)
		h.current = nil
	} else {
		runtime.Call(runtime.Entry(item.entry), runtime.Context(item.context))
		item.done = true
	}
	h.compactJobs()
	return true
}

func (h *Handler) enqueue(item *job) {
	if item == nil || item.done || item.queued {
		return
	}
	item.queued = true
	h.jobs = append(h.jobs, item)
}

func (h *Handler) compactJobs() {
	if h.jobHead != len(h.jobs) {
		return
	}
	h.jobs = h.jobs[:0]
	h.jobHead = 0
}

func (h *Handler) waitFor(wait *waiter) {
	for !wait.done {
		if runtime.StackSupported() && h.current != nil {
			current := h.current
			runtime.StackSwitch(uintptr(unsafe.Pointer(&current.sp)), h.scheduler)
			continue
		}
		if !h.runOne() {
			h.blockForever()
		}
	}
}

func (h *Handler) wake(wait *waiter) {
	if wait != nil && wait.task != nil {
		h.enqueue(wait.task)
	}
}

// complete removes every sibling registration before making a selected task
// runnable, so one select statement can never commit more than one case.
func (h *Handler) complete(wait *waiter, open bool, status int) bool {
	if wait == nil || wait.done {
		return false
	}
	binding := h.selectBinding(wait)
	if binding == nil {
		return false
	}
	group := binding.group
	if group.selected >= 0 {
		return false
	}
	wait.open = open
	wait.status = status
	wait.done = true
	group.selected = binding.index
	group.status = status
	for i := 0; i < len(group.bindings); i++ {
		sibling := group.bindings[i]
		if sibling == nil || sibling == binding || sibling.wait.done {
			continue
		}
		if sibling.send {
			removeWaiter(&sibling.owner.senders, sibling.wait)
		} else {
			removeWaiter(&sibling.owner.receivers, sibling.wait)
		}
		sibling.wait.done = true
	}
	h.removeSelectGroup(group)
	h.wake(wait)
	return true
}

func (h *Handler) selectBinding(wait *waiter) *selectBinding {
	for channelIndex := 0; channelIndex < len(h.channels); channelIndex++ {
		c := h.channels[channelIndex]
		for i := 0; i < len(c.selects); i++ {
			if c.selects[i].wait == wait {
				return c.selects[i]
			}
		}
	}
	return nil
}

func (h *Handler) removeSelectGroup(group *selectWait) {
	for channelIndex := 0; channelIndex < len(h.channels); channelIndex++ {
		bindings := &h.channels[channelIndex].selects
		for i := len(*bindings) - 1; i >= 0; i-- {
			if (*bindings)[i].group != group {
				continue
			}
			copy((*bindings)[i:], (*bindings)[i+1:])
			*bindings = (*bindings)[:len(*bindings)-1]
		}
	}
}

// ChanCreate allocates a channel identified by a stable nonzero table index.
func (h *Handler) ChanCreate(elementSize uintptr, capacity int) uintptr {
	if capacity < 0 {
		return 0
	}
	c := &channel{elementSize: int(elementSize), capacity: capacity}
	if capacity > 0 && c.elementSize > 0 {
		c.buffer = make([]byte, capacity*c.elementSize)
	}
	h.channels = append(h.channels, c)
	return uintptr(len(h.channels))
}

// ChanSend performs a buffered send or serialized rendezvous.
func (h *Handler) ChanSend(handle uintptr, value uintptr) int {
	c := h.channel(handle)
	if c == nil {
		h.blockForever()
		return int(runtime.StatusInvalid)
	}
	if c.closed {
		return int(runtime.StatusClosed)
	}
	if receiver := popWaiter(&c.receivers); receiver != nil {
		copyFromAddress(receiver.target, value, c.elementSize)
		if receiver.status == selectPendingStatus {
			h.complete(receiver, true, int(runtime.StatusOK))
		} else {
			receiver.open = true
			receiver.done = true
			h.wake(receiver)
		}
		return int(runtime.StatusOK)
	}
	if c.length < c.capacity {
		c.pushBuffer(value)
		return int(runtime.StatusOK)
	}
	wait := &waiter{value: bytesFromAddress(value, c.elementSize), status: int(runtime.StatusOK), task: h.current}
	c.senders = append(c.senders, wait)
	h.waitFor(wait)
	return wait.status
}

// ChanReceive performs a buffered receive or serialized rendezvous.
func (h *Handler) ChanReceive(handle uintptr, target uintptr) bool {
	c := h.channel(handle)
	if c == nil {
		h.blockForever()
		return false
	}
	if c.length > 0 {
		c.popBuffer(target)
		h.fillBufferFromSender(c)
		return true
	}
	if sender := popWaiter(&c.senders); sender != nil {
		copyBytesToAddress(target, sender.value)
		if sender.status == selectPendingStatus {
			h.complete(sender, false, int(runtime.StatusOK))
		} else {
			sender.done = true
			h.wake(sender)
		}
		return true
	}
	if c.closed {
		return false
	}
	wait := &waiter{target: target, task: h.current}
	c.receivers = append(c.receivers, wait)
	h.waitFor(wait)
	return wait.open
}

// ChanSelect chooses one ready case atomically with respect to this serialized
// handler. Ready choices rotate to avoid permanently favoring the first case.
func (h *Handler) ChanSelect(cases []runtime.ChanSelectValue, hasDefault bool) (int, int) {
	for {
		ready := make([]int, 0, len(cases))
		for i := 0; i < len(cases); i++ {
			if h.selectReady(cases[i]) {
				ready = append(ready, i)
			}
		}
		if len(ready) > 0 {
			choice := ready[h.selectNext%len(ready)]
			h.selectNext++
			return choice, h.selectApply(cases[choice])
		}
		if hasDefault {
			return -1, int(runtime.StatusOK)
		}
		return h.waitSelect(cases)
	}
}

func (h *Handler) waitSelect(cases []runtime.ChanSelectValue) (int, int) {
	group := &selectWait{bindings: make([]*selectBinding, len(cases)), selected: -1}
	for i := 0; i < len(cases); i++ {
		value := cases[i]
		c := h.channel(uintptr(value.Channel))
		if c == nil {
			continue
		}
		if value.Direction == runtime.SelectSend {
			wait := &waiter{value: bytesFromAddress(value.Value, c.elementSize), status: selectPendingStatus, task: h.current}
			group.bindings[i] = &selectBinding{wait: wait, group: group, index: i, owner: c, send: true}
			c.senders = append(c.senders, wait)
		} else {
			wait := &waiter{target: value.Value, status: selectPendingStatus, task: h.current}
			group.bindings[i] = &selectBinding{wait: wait, group: group, index: i, owner: c}
			c.receivers = append(c.receivers, wait)
		}
		c.selects = append(c.selects, group.bindings[i])
	}
	for group.selected < 0 {
		if runtime.StackSupported() && h.current != nil {
			current := h.current
			runtime.StackSwitch(uintptr(unsafe.Pointer(&current.sp)), h.scheduler)
		} else if !h.runOne() {
			h.blockForever()
		}
	}
	wait := group.bindings[group.selected].wait
	if cases[group.selected].Direction == runtime.SelectReceive && cases[group.selected].ReceiveOK != 0 {
		*(*bool)(unsafe.Pointer(cases[group.selected].ReceiveOK)) = wait.open
	}
	return group.selected, group.status
}

// ChanClose closes a channel and wakes blocked operations.
func (h *Handler) ChanClose(handle uintptr) int {
	c := h.channel(handle)
	if c == nil {
		return int(runtime.StatusInvalid)
	}
	if c.closed {
		return int(runtime.StatusClosed)
	}
	c.closed = true
	for len(c.senders) > 0 {
		wait := popWaiter(&c.senders)
		if wait.status == selectPendingStatus {
			h.complete(wait, false, int(runtime.StatusClosed))
		} else {
			wait.status = int(runtime.StatusClosed)
			wait.done = true
			h.wake(wait)
		}
	}
	for len(c.receivers) > 0 && c.length > 0 {
		wait := popWaiter(&c.receivers)
		c.popBuffer(wait.target)
		if wait.status == selectPendingStatus {
			h.complete(wait, true, int(runtime.StatusOK))
		} else {
			wait.open = true
			wait.done = true
			h.wake(wait)
		}
	}
	for len(c.receivers) > 0 {
		wait := popWaiter(&c.receivers)
		if wait.status == selectPendingStatus {
			h.complete(wait, false, int(runtime.StatusOK))
		} else {
			wait.open = false
			wait.done = true
			h.wake(wait)
		}
	}
	return int(runtime.StatusOK)
}

// ChanLen returns the number of buffered values.
func (h *Handler) ChanLen(handle uintptr) int {
	c := h.channel(handle)
	if c == nil {
		return 0
	}
	return c.length
}

// ChanCap returns the channel buffer capacity.
func (h *Handler) ChanCap(handle uintptr) int {
	c := h.channel(handle)
	if c == nil {
		return 0
	}
	return c.capacity
}

func (h *Handler) channel(handle uintptr) *channel {
	index := int(handle) - 1
	if index < 0 || index >= len(h.channels) {
		return nil
	}
	return h.channels[index]
}

func (h *Handler) selectReady(value runtime.ChanSelectValue) bool {
	c := h.channel(uintptr(value.Channel))
	if c == nil {
		return false
	}
	if value.Direction == runtime.SelectSend {
		return c.closed || len(c.receivers) > 0 || c.length < c.capacity
	}
	return c.length > 0 || len(c.senders) > 0 || c.closed
}

func (h *Handler) selectApply(value runtime.ChanSelectValue) int {
	c := h.channel(uintptr(value.Channel))
	if value.Direction == runtime.SelectSend {
		if c.closed {
			return int(runtime.StatusClosed)
		}
		if receiver := popWaiter(&c.receivers); receiver != nil {
			copyFromAddress(receiver.target, value.Value, c.elementSize)
			if receiver.status == selectPendingStatus {
				h.complete(receiver, true, int(runtime.StatusOK))
			} else {
				receiver.open = true
				receiver.done = true
				h.wake(receiver)
			}
		} else {
			c.pushBuffer(value.Value)
		}
		return int(runtime.StatusOK)
	}
	open := true
	if c.length > 0 {
		c.popBuffer(value.Value)
		h.fillBufferFromSender(c)
	} else if sender := popWaiter(&c.senders); sender != nil {
		copyBytesToAddress(value.Value, sender.value)
		if sender.status == selectPendingStatus {
			h.complete(sender, false, int(runtime.StatusOK))
		} else {
			sender.done = true
			h.wake(sender)
		}
	} else {
		open = false
	}
	if value.ReceiveOK != 0 {
		*(*bool)(unsafe.Pointer(value.ReceiveOK)) = open
	}
	return int(runtime.StatusOK)
}

func (c *channel) pushBuffer(source uintptr) {
	if c.capacity == 0 {
		return
	}
	index := (c.head + c.length) % c.capacity
	copyAddressToBytes(c.buffer[index*c.elementSize:(index+1)*c.elementSize], source)
	c.length++
}

func (c *channel) popBuffer(target uintptr) {
	if c.length == 0 {
		return
	}
	copyBytesToAddress(target, c.buffer[c.head*c.elementSize:(c.head+1)*c.elementSize])
	c.head = (c.head + 1) % c.capacity
	c.length--
}

func (h *Handler) fillBufferFromSender(c *channel) {
	if c.length >= c.capacity {
		return
	}
	sender := popWaiter(&c.senders)
	if sender == nil {
		return
	}
	index := (c.head + c.length) % c.capacity
	copy(c.buffer[index*c.elementSize:(index+1)*c.elementSize], sender.value)
	c.length++
	if sender.status == selectPendingStatus {
		h.complete(sender, false, int(runtime.StatusOK))
	} else {
		sender.done = true
		h.wake(sender)
	}
}

func popWaiter(waiters *[]*waiter) *waiter {
	if len(*waiters) == 0 {
		return nil
	}
	value := (*waiters)[0]
	copy((*waiters)[0:], (*waiters)[1:])
	*waiters = (*waiters)[:len(*waiters)-1]
	return value
}

func removeWaiter(waiters *[]*waiter, wanted *waiter) {
	for i := 0; i < len(*waiters); i++ {
		if (*waiters)[i] != wanted {
			continue
		}
		copy((*waiters)[i:], (*waiters)[i+1:])
		*waiters = (*waiters)[:len(*waiters)-1]
		return
	}
}

func bytesFromAddress(address uintptr, size int) []byte {
	value := make([]byte, size)
	copyAddressToBytes(value, address)
	return value
}

func copyAddressToBytes(target []byte, source uintptr) {
	for i := 0; i < len(target); i++ {
		address := source + uintptr(i)
		target[i] = (*[1]byte)(unsafe.Pointer(address))[0]
	}
}

func copyBytesToAddress(target uintptr, source []byte) {
	for i := 0; i < len(source); i++ {
		address := target + uintptr(i)
		(*[1]byte)(unsafe.Pointer(address))[0] = source[i]
	}
}

func copyFromAddress(target uintptr, source uintptr, size int) {
	for i := 0; i < size; i++ {
		targetAddress := target + uintptr(i)
		sourceAddress := source + uintptr(i)
		(*[1]byte)(unsafe.Pointer(targetAddress))[0] = (*[1]byte)(unsafe.Pointer(sourceAddress))[0]
	}
}

func (h *Handler) blockForever() {
	for {
		if runtime.StackSupported() && h.current != nil {
			current := h.current
			runtime.StackSwitch(uintptr(unsafe.Pointer(&current.sp)), h.scheduler)
			continue
		}
		if h.runOne() {
			continue
		}
	}
}
