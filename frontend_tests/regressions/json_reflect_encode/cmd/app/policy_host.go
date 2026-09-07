//go:build !renvo

package main

// The Go toolchain has no Renvo annotation policy. Its oracle checks encoding
// values; policy enforcement is tested by the native Renvo fixture.
func checkRenvoPolicy() {}
