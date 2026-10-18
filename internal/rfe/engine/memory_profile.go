package engine

import emu "renvo.dev/internal/rfe/runtime"

// Native-exit metadata is diagnostic only: observed tags and permissions do not
// grant access, and no profiling read performs an architectural memory access.
type DispatchMemorySample struct {
	PC          uint64 `json:"p"`
	Count       uint64 `json:"n"`
	Address     uint64 `json:"a"`
	Status      uint64 `json:"s"`
	Permissions uint64 `json:"r"`
	PageMatch   bool   `json:"m"`
}

func (e *Engine) recordMemoryExit(pc uint64, context *emu.MemoryContext) {
	if e.profileMemory == nil {
		e.profileMemory = new([256]DispatchMemorySample)
	}
	first := int(((pc>>1)^(pc>>12))&63) * 4
	slot := first
	for i := first; i < first+4; i++ {
		if e.profileMemory[i].PC == pc && e.profileMemory[i].Status == context.Status && e.profileMemory[i].Count != 0 {
			slot = i
			break
		}
		if e.profileMemory[i].Count < e.profileMemory[slot].Count {
			slot = i
		}
	}
	s := &e.profileMemory[slot]
	if s.PC != pc || s.Status != context.Status || s.Count == 0 {
		*s = DispatchMemorySample{PC: pc, Status: context.Status}
	}
	s.Count++
	s.Address = context.Address
	page := *context.PageSlot(context.Address >> 12)
	s.PageMatch = page.Data != nil && page.Number == context.Address>>12
	s.Permissions = 0
	if s.PageMatch {
		s.Permissions = page.Permissions
	}
}

func (e *Engine) memoryExitProfile() []DispatchMemorySample {
	if e.profileMemory == nil {
		return nil
	}
	top := make([]DispatchMemorySample, 0, 8)
	for _, s := range e.profileMemory {
		if s.Count == 0 {
			continue
		}
		at := 0
		for at < len(top) && top[at].Count >= s.Count {
			at++
		}
		if at >= 8 {
			continue
		}
		if len(top) < 8 {
			top = append(top, DispatchMemorySample{})
		}
		for i := len(top) - 1; i > at; i-- {
			top[i] = top[i-1]
		}
		top[at] = s
	}
	return top
}

// MemoryExitProfile returns at most eight retained hot native-exit samples.
// Disabled profiling records no samples and returns nil.
func (e *Engine) MemoryExitProfile() []DispatchMemorySample { return e.memoryExitProfile() }
