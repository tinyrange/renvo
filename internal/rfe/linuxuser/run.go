package linuxuser

import (
	"fmt"
	"io"
	"renvo.dev/internal/rfe/engine"
)

// SyscallABI describes only guest register/trap conventions. Syscall services
// operate on values and checked guest memory, never a particular CPU struct.
type SyscallABI struct {
	Number, Result int
	Arguments      [6]int
	Trap           func(error) (bool, error)
}

func (a SyscallABI) valid(words int) bool {
	if a.Trap == nil || a.Number < 0 || a.Number >= words || a.Result < 0 || a.Result >= words {
		return false
	}
	for _, i := range a.Arguments {
		if i < 0 || i >= words {
			return false
		}
	}
	return true
}
func (p *Process) ApplySyscall(state []uint64, abi SyscallABI, output, diagnostics io.Writer) (bool, int, error) {
	if !abi.valid(len(state)) {
		return false, 0, fmt.Errorf("invalid syscall register ABI")
	}
	args := [6]uint64{}
	for i, slot := range abi.Arguments {
		args[i] = state[slot]
	}
	value, exited, code := p.Syscall(state[abi.Number], args, output, diagnostics)
	if !exited {
		state[abi.Result] = value
	}
	return exited, code, nil
}

// Run preserves an absolute retirement ceiling across resumptions. Architecture
// adapters own trap recognition and register conventions; the engine owns tiers.
func (p *Process) Run(cpu engine.CPU, arch engine.Architecture, abi SyscallABI, config engine.EngineConfig, limit uint64, output, diagnostics io.Writer) (code int, stats engine.Stats, err error) {
	if !abi.valid(arch.StateWords) {
		return 0, stats, fmt.Errorf("invalid syscall ABI")
	}
	e, err := engine.New(config, arch)
	if err != nil {
		return 0, stats, err
	}
	defer func() {
		stats = e.Stats
		closeErr := e.Close()
		if err == nil {
			err = closeErr
		}
	}()
	for *cpu.Retirement() < limit {
		err = e.RunQuanta(cpu, limit-*cpu.Retirement())
		if err != nil {
			call, trapErr := abi.Trap(err)
			if trapErr != nil {
				return 0, stats, trapErr
			}
			if call {
				exited, status, callErr := p.ApplySyscall(cpu.Registers(), abi, output, diagnostics)
				if callErr != nil {
					return 0, stats, callErr
				}
				if exited {
					return status, stats, nil
				}
				err = nil
			}
		}
		if err != nil {
			return 0, stats, err
		}
	}
	return 0, stats, fmt.Errorf("instruction budget exhausted after %d instructions", *cpu.Retirement())
}
