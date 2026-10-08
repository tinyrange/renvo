//go:build !renvo && !amd64 && !arm64

package runimage

func callPure(entry, state, stackTop uintptr) { panic("unsupported native RFE host") }

func callContext(entry, state, context, stackTop uintptr) { panic("unsupported native RFE host") }
