package engine

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
