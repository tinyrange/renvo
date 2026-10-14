package engine

// NativeDeadlineCPU permits a host-owned foreign session to poll the same
// real monotonic oscillator every <=64 instructions without returning to Go.
// A zero deadline retains conservative scheduling. Expiry is not a guest fault
// or permission grant; it returns before another architectural operation.
type NativeDeadlineCPU interface {
	NativeDeadline() uint64
	NativeDeadlineExpired()
}

// TranslationClock optionally pauses the VM oscillator while the vCPU is
// stopped for decoding/compilation or arena reclamation. Guest execution and
// memory/device operations must be outside these scopes. CPUs without this
// capability retain their original behavior.
type TranslationClock interface {
	PauseTranslationClock()
	ResumeTranslationClock()
}

func pauseTranslationClock(c CPU) func() {
	if clock, ok := c.(TranslationClock); ok {
		clock.PauseTranslationClock()
		return clock.ResumeTranslationClock
	}
	return func() {}
}
