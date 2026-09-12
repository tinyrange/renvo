//go:build renvo

package runtime

import "fmt"

// The portable IR evaluator is available to Renvo-built hosts. Native code
// mapping currently uses the Go-hosted runner's OS adapters.
type Native struct{ Bytes int }

func NewNative(capacity int) (*Native, error) {
	return nil, fmt.Errorf("native RFE mapping requires the Go-hosted runner")
}
func (n *Native) Compile(ops []Op, words int) (int, error) {
	return 0, fmt.Errorf("native mapping unavailable")
}
func (n *Native) Call(entry int, state []uint64) error {
	return fmt.Errorf("native mapping unavailable")
}
func (n *Native) Close() error { return nil }
