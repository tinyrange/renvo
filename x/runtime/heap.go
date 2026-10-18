package runtime

// Collect runs a synchronous collection on desktop-heap targets. Live objects
// do not move. Arena-only targets perform no work. Collection may retain dead
// objects when a word conservatively resembles a live heap pointer.
func Collect() { renvo_runtime_HeapCollect() }

func renvo_runtime_HeapCollect() {}
