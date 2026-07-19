package main

type EventHandler func()

type Base struct {
	Event EventHandler
}

type Derived struct {
	Base
	hits int
}

func newDerived() *Derived { return &Derived{} }

func (derived *Derived) handle() { derived.hits++ }

type Viewer struct{}

func newViewer() *Viewer { return &Viewer{} }

func (viewer *Viewer) Event() int { return 41 }

func main() {
	derived := newDerived()
	if derived.Event != nil {
		print("FAIL\n")
		return
	}
	derived.Event = derived.handle
	derived.Event()
	viewer := newViewer()
	if derived.hits != 1 || viewer.Event()+1 != 42 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
	return
}
