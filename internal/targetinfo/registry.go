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
