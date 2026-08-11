package gofont

import "embed"

// Keep the two fonts in one compressed archive. Besides reducing application
// size, this avoids expanding hundreds of kilobytes of binary data into Go
// string-literal tokens while the self-hosted frontend loads this package.
//
//go:embed Go-Regular.ttf Go-Mono.ttf
var fontFiles embed.FS

var regularBytes []byte
var monoBytes []byte

func regularData() []byte {
	if len(regularBytes) == 0 {
		regularBytes, _ = fontFiles.ReadFile("Go-Regular.ttf")
	}
	return regularBytes
}

func monoData() []byte {
	if len(monoBytes) == 0 {
		monoBytes, _ = fontFiles.ReadFile("Go-Mono.ttf")
	}
	return monoBytes
}
