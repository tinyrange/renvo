// Package targetinfo owns Renvo's canonical target descriptors.
package targetinfo

//go:generate go run ./cmd/gentargets

type Descriptor struct {
	Name                string
	Backend             string
	Aliases             []string
	OS                  string
	ISA                 string
	WordBits            int
	PointerBits         int
	CodePointerBits     int
	FunctionPointerBits int
	MaxAlign            int
	Endian              string
	ABI                 string
	Image               string
	Runtime             []string
	Tags                []string
	Capabilities        []string
	Definition          [32]byte
	DescriptorVersion   int
	Advertised          bool
	Virtual             bool
	DefaultArena        int
	ReleaseArtifact     string
	IDE                 bool
}

// Representative returns the built-in semantic adapter used to bootstrap a
// target described by external machine facts. The definition still owns code
// generation; this selects only the compatible shared frontend/backend kernel.
func Representative(osName string, isa string, wordBits int, capabilities []string) string {
	if osName == "windows" {
		if isa == "aarch64" || isa == "arm64" {
			return "windows/arm64"
		}
		if wordBits == 32 {
			return "windows/386"
		}
		return "windows/amd64"
	}
	if osName == "darwin" {
		return "darwin/arm64"
	}
	if osName == "wasi" || osName == "browser" {
		return "wasi/wasm32"
	}
	if osName == "vm" {
		return "vm/vm32"
	}
	if descriptorHas(capabilities, "kernel_module") {
		return "linux-kernel/amd64"
	}
	if isa == "aarch64" || isa == "arm64" {
		return "linux/aarch64"
	}
	if isa == "arm" {
		return "linux/arm"
	}
	if wordBits == 32 {
		return "linux/386"
	}
	return "linux/amd64"
}

func descriptorHas(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}
