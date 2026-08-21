package serial

import (
	"testing"
	"time"
	"unsafe"

	runtime "renvo.dev/x/runtime"
)

func TestMonotonicTimerWakeAndStop(t *testing.T) {
	h := New()
	started := time.Now()
	timer := h.TimerStart(int64(time.Millisecond))
	if !h.TimerWait(timer) {
		t.Fatal("timer did not fire")
	}
	if time.Since(started) < time.Millisecond {
		t.Fatal("timer fired before its deadline")
	}
	stopped := h.TimerStart(int64(time.Hour))
	if !h.TimerStop(stopped) || h.TimerWait(stopped) || h.TimerStop(stopped) {
		t.Fatal("stopped timer state is inconsistent")
	}
}

func TestBufferedChannelAndSelect(t *testing.T) {
	h := New()
	channel := h.ChanCreate(unsafe.Sizeof(int(0)), 2)
	first, second := 7, 9
	if h.ChanSend(channel, unsafe.Pointer(&first)) != int(runtime.StatusOK) || h.ChanSend(channel, unsafe.Pointer(&second)) != int(runtime.StatusOK) {
		t.Fatal("buffered send failed")
	}
	if h.ChanLen(channel) != 2 || h.ChanCap(channel) != 2 {
		t.Fatalf("len/cap = %d/%d", h.ChanLen(channel), h.ChanCap(channel))
	}
	var got int
	var open bool
	cases := []runtime.ChanSelectValue{{Channel: runtime.Channel(channel), Value: unsafe.Pointer(&got), ReceiveOK: unsafe.Pointer(&open), Direction: runtime.SelectReceive}}
	index, status := h.ChanSelect(cases, false)
	if index != 0 || status != int(runtime.StatusOK) || got != 7 || !open {
		t.Fatalf("select = index %d status %d value %d open %v", index, status, got, open)
	}
	if h.ChanClose(channel) != int(runtime.StatusOK) {
		t.Fatal("close failed")
	}
	got = 0
	if !h.ChanReceive(channel, unsafe.Pointer(&got)) || got != 9 {
		t.Fatalf("drain = %d", got)
	}
	if h.ChanReceive(channel, unsafe.Pointer(&got)) {
		t.Fatal("closed channel reported open")
	}
}

func TestSelectDefaultAndClosedSend(t *testing.T) {
	h := New()
	channel := h.ChanCreate(1, 0)
	value := byte(3)
	cases := []runtime.ChanSelectValue{{Channel: runtime.Channel(channel), Value: unsafe.Pointer(&value), Direction: runtime.SelectReceive}}
	index, status := h.ChanSelect(cases, true)
	if index != -1 || status != int(runtime.StatusOK) {
		t.Fatalf("default = %d/%d", index, status)
	}
	h.ChanClose(channel)
	cases[0].Direction = runtime.SelectSend
	index, status = h.ChanSelect(cases, false)
	if index != 0 || status != int(runtime.StatusClosed) {
		t.Fatalf("closed send = %d/%d", index, status)
	}
}

func TestSelectCompletionAtomicallyRemovesSiblingWaiters(t *testing.T) {
	h := New()
	first := h.ChanCreate(unsafe.Sizeof(int(0)), 0)
	second := h.ChanCreate(unsafe.Sizeof(int(0)), 0)
	firstChannel := h.channel(first)
	secondChannel := h.channel(second)
	group := &selectWait{bindings: make([]*selectBinding, 2), selected: -1}
	for index, owner := range []*channel{firstChannel, secondChannel} {
		wait := &waiter{target: unsafe.Pointer(new(int)), status: selectPendingStatus}
		group.bindings[index] = &selectBinding{wait: wait, group: group, index: index, owner: owner}
		owner.selects = append(owner.selects, group.bindings[index])
		owner.receivers = append(owner.receivers, wait)
	}
	value := 41
	if status := h.ChanSend(first, unsafe.Pointer(&value)); status != int(runtime.StatusOK) {
		t.Fatalf("send status = %d", status)
	}
	if group.selected != 0 || len(firstChannel.receivers) != 0 || len(secondChannel.receivers) != 0 {
		t.Fatalf("selection = %d, remaining waiters = %d/%d", group.selected, len(firstChannel.receivers), len(secondChannel.receivers))
	}
	if h.complete(group.bindings[1].wait, true, int(runtime.StatusOK)) {
		t.Fatal("a second select case completed")
	}
}

func TestSelectRotatesAmongReadyCases(t *testing.T) {
	h := New()
	channels := []uintptr{h.ChanCreate(unsafe.Sizeof(int(0)), 1), h.ChanCreate(unsafe.Sizeof(int(0)), 1)}
	values := []int{10, 20}
	for i := 0; i < len(channels); i++ {
		if status := h.ChanSend(channels[i], unsafe.Pointer(&values[i])); status != int(runtime.StatusOK) {
			t.Fatalf("initial send %d status = %d", i, status)
		}
	}
	for iteration := 0; iteration < 4; iteration++ {
		var got int
		cases := []runtime.ChanSelectValue{
			{Channel: runtime.Channel(channels[0]), Value: unsafe.Pointer(&got), Direction: runtime.SelectReceive},
			{Channel: runtime.Channel(channels[1]), Value: unsafe.Pointer(&got), Direction: runtime.SelectReceive},
		}
		index, status := h.ChanSelect(cases, false)
		want := iteration % 2
		if status != int(runtime.StatusOK) || index != want || got != values[want] {
			t.Fatalf("iteration %d = index %d status %d value %d", iteration, index, status, got)
		}
		if status := h.ChanSend(channels[want], unsafe.Pointer(&values[want])); status != int(runtime.StatusOK) {
			t.Fatalf("refill %d status = %d", want, status)
		}
	}
}

type fuzzChannelModel struct {
	values   []uint64
	capacity int
	closed   bool
}

func FuzzHandlerChannelStateMachine(f *testing.F) {
	f.Add([]byte{2, 0, 0, 1, 0, 3, 1, 4, 0, 2})
	f.Add([]byte{4, 3, 1, 4, 2, 3, 0, 1, 5, 2, 0, 1, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		if len(data) > 4096 {
			data = data[:4096]
		}
		count := 2 + int(data[0]%3)
		handler := New()
		handles := make([]uintptr, count)
		models := make([]fuzzChannelModel, count)
		for i := 0; i < count; i++ {
			capacity := 1 + int(data[(i+1)%len(data)]%4)
			handles[i] = handler.ChanCreate(unsafe.Sizeof(uint64(0)), capacity)
			models[i].capacity = capacity
		}
		for step := 1; step < len(data); step++ {
			index := int(data[step]>>3) % count
			model := &models[index]
			value := uint64(step)<<32 | uint64(data[step])
			switch data[step] % 6 {
			case 0:
				if model.closed {
					if status := handler.ChanSend(handles[index], unsafe.Pointer(&value)); status != int(runtime.StatusClosed) {
						t.Fatalf("step %d: closed send status = %d", step, status)
					}
				} else if len(model.values) < model.capacity {
					if status := handler.ChanSend(handles[index], unsafe.Pointer(&value)); status != int(runtime.StatusOK) {
						t.Fatalf("step %d: send status = %d", step, status)
					}
					model.values = append(model.values, value)
				}
			case 1:
				if len(model.values) > 0 {
					var got uint64
					if !handler.ChanReceive(handles[index], unsafe.Pointer(&got)) || got != model.values[0] {
						t.Fatalf("step %d: receive = %d, want %d", step, got, model.values[0])
					}
					model.values = model.values[1:]
				} else if model.closed {
					var got uint64
					if handler.ChanReceive(handles[index], unsafe.Pointer(&got)) {
						t.Fatalf("step %d: closed empty receive reported open", step)
					}
				}
			case 2:
				status := handler.ChanClose(handles[index])
				want := runtime.StatusOK
				if model.closed {
					want = runtime.StatusClosed
				} else {
					model.closed = true
				}
				if status != int(want) {
					t.Fatalf("step %d: close status = %d, want %d", step, status, want)
				}
			case 3:
				fuzzReceiveSelect(t, handler, handles, models, index, (index+1)%count, step)
			case 4:
				fuzzSendSelect(t, handler, handles, models, index, (index+1)%count, value, step)
			case 5:
				if got := handler.ChanLen(handles[index]); got != len(model.values) {
					t.Fatalf("step %d: len = %d, want %d", step, got, len(model.values))
				}
				if got := handler.ChanCap(handles[index]); got != model.capacity {
					t.Fatalf("step %d: cap = %d, want %d", step, got, model.capacity)
				}
			}
		}
	})
}

func fuzzReceiveSelect(t *testing.T, handler *Handler, handles []uintptr, models []fuzzChannelModel, first int, second int, step int) {
	t.Helper()
	var got uint64
	var open bool
	cases := []runtime.ChanSelectValue{
		{Channel: runtime.Channel(handles[first]), Value: unsafe.Pointer(&got), ReceiveOK: unsafe.Pointer(&open), Direction: runtime.SelectReceive},
		{Channel: runtime.Channel(handles[second]), Value: unsafe.Pointer(&got), ReceiveOK: unsafe.Pointer(&open), Direction: runtime.SelectReceive},
	}
	index, status := handler.ChanSelect(cases, true)
	if status != int(runtime.StatusOK) {
		t.Fatalf("step %d: receive select status = %d", step, status)
	}
	ready := []bool{len(models[first].values) > 0 || models[first].closed, len(models[second].values) > 0 || models[second].closed}
	if index < 0 {
		if ready[0] || ready[1] {
			t.Fatalf("step %d: receive select chose default with ready case", step)
		}
		return
	}
	if index > 1 || !ready[index] {
		t.Fatalf("step %d: receive select index = %d, ready = %v", step, index, ready)
	}
	selected := first
	if index == 1 {
		selected = second
	}
	model := &models[selected]
	if len(model.values) == 0 {
		if open {
			t.Fatalf("step %d: closed receive reported open", step)
		}
		return
	}
	if !open || got != model.values[0] {
		t.Fatalf("step %d: selected receive = %d/%v, want %d/true", step, got, open, model.values[0])
	}
	model.values = model.values[1:]
}

func fuzzSendSelect(t *testing.T, handler *Handler, handles []uintptr, models []fuzzChannelModel, first int, second int, value uint64, step int) {
	t.Helper()
	cases := []runtime.ChanSelectValue{
		{Channel: runtime.Channel(handles[first]), Value: unsafe.Pointer(&value), Direction: runtime.SelectSend},
		{Channel: runtime.Channel(handles[second]), Value: unsafe.Pointer(&value), Direction: runtime.SelectSend},
	}
	index, status := handler.ChanSelect(cases, true)
	ready := []bool{models[first].closed || len(models[first].values) < models[first].capacity, models[second].closed || len(models[second].values) < models[second].capacity}
	if index < 0 {
		if status != int(runtime.StatusOK) || ready[0] || ready[1] {
			t.Fatalf("step %d: send select default = %d/%d, ready = %v", step, index, status, ready)
		}
		return
	}
	if index > 1 || !ready[index] {
		t.Fatalf("step %d: send select index = %d, ready = %v", step, index, ready)
	}
	selected := first
	if index == 1 {
		selected = second
	}
	model := &models[selected]
	if model.closed {
		if status != int(runtime.StatusClosed) {
			t.Fatalf("step %d: selected closed send status = %d", step, status)
		}
		return
	}
	if status != int(runtime.StatusOK) {
		t.Fatalf("step %d: selected send status = %d", step, status)
	}
	model.values = append(model.values, value)
}
