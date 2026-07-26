// Package structs defines marker types that modify struct properties.
package structs

// HostLayout marks a struct as using the target platform's host memory layout,
// generally following its C ABI.
//
// By convention HostLayout is the type of a blank field placed first in the
// struct definition:
//
//	type Header struct {
//		_ structs.HostLayout
//		Kind uint8
//		Size uint32
//	}
type HostLayout struct {
	_ hostLayout `r:"h"`
}

type hostLayout struct{}
