package engine

import (
	"encoding/binary"
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"renvo.dev/internal/rfe/systemclock"
	"testing"
	"time"
)

type clockCPU struct {
	stubCPU
	clock   *systemclock.Clock
	depth   int
	elapsed *time.Duration
}

func (c *clockCPU) PauseTranslationClock()  { c.depth++; c.clock.Pause() }
func (c *clockCPU) ResumeTranslationClock() { c.clock.Resume(); c.depth-- }
func (c *clockCPU) Step() error {
	if c.depth != 0 {
		return fmt.Errorf("guest interpreted with paused oscillator")
	}
	*c.elapsed += time.Millisecond
	return c.stubCPU.Step()
}
func (c *clockCPU) ExecuteInstruction(ins Instruction) error {
	if c.depth != 0 {
		return fmt.Errorf("guest cached instruction with paused oscillator")
	}
	*c.elapsed += time.Millisecond
	return c.stubCPU.ExecuteInstruction(ins)
}

func TestTranslationClockExcludesColdWorkAndResumesOnFallback(t *testing.T) {
	for _, reject := range []bool{false, true} {
		elapsed := 100 * time.Millisecond
		clock := systemclock.WithSource(100000000, func() time.Duration { return elapsed }, func(time.Duration) { t.Fatal("unexpected wait") })
		memory := new(stubMemory)
		binary.LittleEndian.PutUint16(memory.bytes[:], 0x101)
		c := &clockCPU{stubCPU: stubCPU{memory: memory}, clock: clock, elapsed: &elapsed}
		arch := stubArchitecture()
		fetch, lower := arch.Fetch, arch.Lower
		arch.Fetch = func(m Memory, pc uint64) (Instruction, error) {
			if c.depth == 0 {
				t.Fatal("cold decode with running vCPU clock")
			}
			elapsed += time.Second
			return fetch(m, pc)
		}
		arch.Lower = func(b *emu.Builder, ins Instruction, completed int) bool {
			if c.depth == 0 {
				t.Fatal("compiler with running vCPU clock")
			}
			elapsed += time.Second
			return !reject && lower(b, ins, completed)
		}
		cfg := DefaultEngineConfig()
		cfg.Mode, cfg.MaxInstructions, cfg.IRThreshold = "ir", 1, 1
		e, err := New(cfg, arch)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Step(c, 1); err != nil {
			t.Fatal(err)
		}
		if c.depth != 0 || c.retired != 1 || c.state[0] != 1 {
			t.Fatal("translation changed progress or left clock stopped", c.depth, c.retired, c.state)
		}
		want := uint64(10000000)
		if reject {
			want += 100000
		} // Actual guest instruction time counts.
		if got := clock.Read(); got != want {
			t.Fatal("translation inflated guest time", reject, got, want)
		}
		elapsed += time.Millisecond
		if clock.Read() != want+100000 {
			t.Fatal("clock did not resume after engine return")
		}
		e.Close()
	}
}
