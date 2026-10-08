//go:build renvo

package runtime

import "fmt"

// The portable IR evaluator is available to Renvo-built hosts. Native code
// mapping currently uses the Go-hosted runner's OS adapters.
type Native struct {
	Bytes        int
	SymbolErrors uint64
}

func NewNative(capacity int) (*Native, error) {
	return nil, fmt.Errorf("native RFE mapping requires the Go-hosted runner")
}
func (n *Native) Compile(ops []Op, words int) (int, error) {
	return 0, fmt.Errorf("native mapping unavailable")
}
func (n *Native) Call(entry int, state []uint64) error {
	return fmt.Errorf("native mapping unavailable")
}
func (n *Native) CallBatch(state []uint64, limit int, next func(int) (int, bool, error)) (int, error) {
	return 0, fmt.Errorf("native mapping unavailable")
}
func (n *Native) Close() error { return nil }

func (n *Native) CompileMemory(ops []Op, words int) (int, error) {
	return 0, fmt.Errorf("native mapping unavailable")
}
func (n *Native) CallMemory(entry int, state []uint64, memory *MemoryContext) error {
	return fmt.Errorf("native mapping unavailable")
}
func (n *Native) CallBatchMemory(state []uint64, memory *MemoryContext, limit int, next func(int) (int, bool, error)) (int, error) {
	return 0, fmt.Errorf("native mapping unavailable")
}

func (n *Native) PrepareLinks(words, pc int) error { return fmt.Errorf("native mapping unavailable") }
func (n *Native) CallLinked(state []uint64, memory *MemoryContext, budget uint64) error {
	return fmt.Errorf("native mapping unavailable")
}

func (n *Native) EnablePerfMap() error                   { return nil }
func (n *Native) NameEntry(entry int, name string) error { return nil }

func (n *Native) AdmitLink(entry, words, instructions int) error {
	return fmt.Errorf("native mapping unavailable")
}

func (n *Native) RunLinkedQuanta(state []uint64, memory *MemoryContext, limit int, remaining, generation uint64) (int, error) {
	return 0, fmt.Errorf("native mapping unavailable")
}

func (n *Native) PrepareTargetLink(pc uint64, entry, words, instructions int) error {
	return fmt.Errorf("native mapping unavailable")
}

func (n *Native) CompileLoop(ops []Op, words, instructions int) (int, error) {
	return 0, fmt.Errorf("native mapping unavailable")
}

const NativeSessionLimit = 65536

func NativeSessionsAvailable() bool { return false }
func (n *Native) RunLinkedSession(state []uint64, memory *MemoryContext, budget uint64) error {
	return fmt.Errorf("native mapping unavailable")
}
