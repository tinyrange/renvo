package board

import "renvo.dev/internal/arena"

func init() {
	// Keep application objects between the two PSRAM framebuffers.
	arena.Reset(0x48800000)
	arena.PersistReset(0x49f00000)
}
