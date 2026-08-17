package runtime

import "testing"

type recordingHandler struct {
	spawnEntry   Entry
	spawnContext Context
	createdSize  uintptr
	createdCap   int
	channel      Channel
	sendStatus   Status
	closeStatus  Status
}

func (h *recordingHandler) Spawn(entry uintptr, context uintptr) {
	h.spawnEntry = Entry(entry)
	h.spawnContext = Context(context)
}

func (h *recordingHandler) ChanCreate(size uintptr, capacity int) uintptr {
	h.createdSize = size
	h.createdCap = capacity
	return uintptr(h.channel)
}

func (h *recordingHandler) ChanSend(uintptr, uintptr) int     { return int(h.sendStatus) }
func (h *recordingHandler) ChanReceive(uintptr, uintptr) bool { return true }
func (h *recordingHandler) ChanSelect([]ChanSelectValue, bool) (int, int) {
	return 2, int(StatusOK)
}
func (h *recordingHandler) ChanClose(uintptr) int { return int(h.closeStatus) }
func (h *recordingHandler) ChanLen(uintptr) int   { return 3 }
func (h *recordingHandler) ChanCap(uintptr) int   { return 5 }

func withHandler(t *testing.T, handler GoHandler) {
	t.Helper()
	previous := activeHandler
	activeHandler = handler
	t.Cleanup(func() { activeHandler = previous })
}

func TestRuntimeWrappersDelegateToHandler(t *testing.T) {
	handler := &recordingHandler{channel: 17, sendStatus: StatusOK, closeStatus: StatusOK}
	withHandler(t, handler)

	renvo_runtime_Spawn(4, 9)
	if handler.spawnEntry != 4 || handler.spawnContext != 9 {
		t.Fatalf("spawn = %d/%d", handler.spawnEntry, handler.spawnContext)
	}
	if channel := renvo_runtime_ChanCreate(12, 7); channel != 17 || handler.createdSize != 12 || handler.createdCap != 7 {
		t.Fatalf("create = %d, size/cap = %d/%d", channel, handler.createdSize, handler.createdCap)
	}
	if !renvo_runtime_ChanReceive(17, 100) || renvo_runtime_ChanLen(17) != 3 || renvo_runtime_ChanCap(17) != 5 {
		t.Fatal("receive/len/cap delegation failed")
	}
	if index := renvo_runtime_ChanSelect(nil, true); index != 2 {
		t.Fatalf("select index = %d", index)
	}
	renvo_runtime_ChanSend(17, 100)
	renvo_runtime_ChanClose(17)
}

func TestRuntimeWrappersReportUsefulPanics(t *testing.T) {
	tests := []struct {
		name string
		call func()
	}{
		{name: "missing handler", call: func() { activeHandler = nil; renvo_runtime_Spawn(1, 1) }},
		{name: "negative capacity", call: func() { renvo_runtime_ChanCreate(1, -1) }},
		{name: "closed send", call: func() { activeHandler = &recordingHandler{sendStatus: StatusClosed}; renvo_runtime_ChanSend(1, 1) }},
		{name: "closed close", call: func() { activeHandler = &recordingHandler{closeStatus: StatusClosed}; renvo_runtime_ChanClose(1) }},
		{name: "nil close", call: func() { activeHandler = &recordingHandler{closeStatus: StatusInvalid}; renvo_runtime_ChanClose(0) }},
	}
	previous := activeHandler
	t.Cleanup(func() { activeHandler = previous })
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deferred := false
			func() {
				defer func() { deferred = recover() != nil }()
				test.call()
			}()
			if !deferred {
				t.Fatal("call did not panic")
			}
		})
	}
}
