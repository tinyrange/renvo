package gofont

import _ "embed"

//go:embed Go-Regular.ttf
var regularBytes []byte

//go:embed Go-Mono.ttf
var monoBytes []byte

func regularData() []byte {
	return regularBytes
}

func monoData() []byte {
	return monoBytes
}
