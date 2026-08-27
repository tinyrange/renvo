// Package rtgb encodes prepared, single-target Renvo backend artifacts.
//
// The codec is deliberately filesystem-free. Preparation policy, cache paths,
// and atomic publication belong to the host adapter.
package rtgb

import (
	"renvo.dev/internal/rbe"
	"renvo.dev/internal/rtg"
)

const (
	Version    = 5
	HeaderSize = 80
	MaxPayload = 64 << 20
	MaxMeta    = 20 << 20
)

const Magic = "RTGB"

type Artifact struct {
	Descriptor   rtg.TargetDescriptor
	Host         string
	Generator    int
	Kernel       int
	Protocol     int
	Unit         int
	Optimization int
	// Enablement identifies the complete RBE source, including library files.
	// Definition continues to identify only backend semantics for unit binding.
	Enablement [32]byte
	// DefinitionFiles preserve the root and imported RTG sources used to
	// prepare this artifact. They let a distributed .rtgb compile project
	// RTGASM without consulting the original source tree.
	DefinitionFiles []rtg.ImportSource
	LibraryFiles    []rbe.File
	Payload         []byte
}

func Encode(artifact Artifact) ([]byte, bool) {
	if !validArtifact(artifact) {
		return nil, false
	}
	var metadata []byte
	metadata = appendString(metadata, artifact.Descriptor.Name)
	metadata = appendString(metadata, artifact.Descriptor.Family)
	metadata = appendString(metadata, artifact.Descriptor.OS)
	metadata = appendString(metadata, artifact.Descriptor.ISA)
	metadata = appendString(metadata, artifact.Descriptor.Endian)
	metadata = appendString(metadata, artifact.Descriptor.ABI)
	metadata = appendString(metadata, artifact.Descriptor.Runtime)
	metadata = appendString(metadata, artifact.Descriptor.OutputKind)
	metadata = appendString(metadata, artifact.Descriptor.Executable)
	metadata = appendString(metadata, artifact.Descriptor.Object)
	metadata = appendString(metadata, artifact.Host)
	metadata = appendStrings(metadata, artifact.Descriptor.Aliases)
	metadata = appendStrings(metadata, artifact.Descriptor.BuildTags)
	metadata = appendStrings(metadata, artifact.Descriptor.Capabilities)
	metadata = appendStrings(metadata, artifact.Descriptor.RuntimeOps)
	metadata = appendVarint(metadata, len(artifact.DefinitionFiles))
	for i := 0; i < len(artifact.DefinitionFiles); i++ {
		metadata = appendString(metadata, artifact.DefinitionFiles[i].Filename)
		metadata = appendBytes(metadata, artifact.DefinitionFiles[i].Source)
	}
	metadata = appendBytes(metadata, artifact.Enablement[:])
	metadata = appendVarint(metadata, len(artifact.LibraryFiles))
	for i := 0; i < len(artifact.LibraryFiles); i++ {
		metadata = appendString(metadata, artifact.LibraryFiles[i].Path)
		metadata = appendBytes(metadata, artifact.LibraryFiles[i].Source)
	}
	if len(metadata) > MaxMeta {
		return nil, false
	}
	out := make([]byte, HeaderSize, HeaderSize+len(metadata)+len(artifact.Payload))
	copy(out, Magic)
	put16(out, 4, Version)
	put16(out, 6, HeaderSize)
	put16(out, 8, artifact.Descriptor.Version)
	put16(out, 10, artifact.Generator)
	put16(out, 12, artifact.Kernel)
	put16(out, 14, artifact.Descriptor.WordBits)
	put16(out, 16, artifact.Descriptor.PointerBits)
	put16(out, 18, artifact.Protocol)
	put16(out, 20, artifact.Unit)
	put16(out, 22, artifact.Optimization)
	put32(out, 24, len(metadata))
	put32(out, 28, len(artifact.Payload))
	copy(out[36:68], artifact.Descriptor.Definition[:])
	put16(out, 68, artifact.Descriptor.CodePointerBits)
	put16(out, 70, artifact.Descriptor.FunctionPointerBits)
	put16(out, 72, artifact.Descriptor.MaxAlign)
	put32(out, 76, artifact.Descriptor.ArenaDefault)
	out = append(out, metadata...)
	out = append(out, artifact.Payload...)
	a, b := checksum(out[HeaderSize:])
	put16(out, 32, a)
	put16(out, 34, b)
	return out, true
}

func Decode(source []byte) (Artifact, bool) {
	var artifact Artifact
	if len(source) < HeaderSize || string(source[:4]) != Magic ||
		read16(source, 4) != Version || read16(source, 6) != HeaderSize {
		return artifact, false
	}
	metadataSize := read32(source, 24)
	payloadSize := read32(source, 28)
	if metadataSize < 0 || metadataSize > MaxMeta || payloadSize <= 0 || payloadSize > MaxPayload ||
		HeaderSize+metadataSize < HeaderSize ||
		HeaderSize+metadataSize+payloadSize != len(source) {
		return artifact, false
	}
	a, b := checksum(source[HeaderSize:])
	if read16(source, 32) != a || read16(source, 34) != b {
		return artifact, false
	}
	artifact.Descriptor.Version = read16(source, 8)
	artifact.Generator = read16(source, 10)
	artifact.Kernel = read16(source, 12)
	artifact.Descriptor.WordBits = read16(source, 14)
	artifact.Descriptor.PointerBits = read16(source, 16)
	artifact.Protocol = read16(source, 18)
	artifact.Unit = read16(source, 20)
	artifact.Optimization = read16(source, 22)
	copy(artifact.Descriptor.Definition[:], source[36:68])
	artifact.Descriptor.CodePointerBits = read16(source, 68)
	artifact.Descriptor.FunctionPointerBits = read16(source, 70)
	artifact.Descriptor.MaxAlign = read16(source, 72)
	artifact.Descriptor.ArenaDefault = read32(source, 76)
	reader := metadataReader{source: source[HeaderSize : HeaderSize+metadataSize], ok: true}
	artifact.Descriptor.Name = reader.string()
	artifact.Descriptor.Family = reader.string()
	artifact.Descriptor.OS = reader.string()
	artifact.Descriptor.ISA = reader.string()
	artifact.Descriptor.Endian = reader.string()
	artifact.Descriptor.ABI = reader.string()
	artifact.Descriptor.Runtime = reader.string()
	artifact.Descriptor.OutputKind = reader.string()
	artifact.Descriptor.Executable = reader.string()
	artifact.Descriptor.Object = reader.string()
	artifact.Host = reader.string()
	artifact.Descriptor.Aliases = reader.strings()
	artifact.Descriptor.BuildTags = reader.strings()
	artifact.Descriptor.Capabilities = reader.strings()
	artifact.Descriptor.RuntimeOps = reader.strings()
	definitionCount := reader.integer()
	if definitionCount < 0 || definitionCount > 4096 {
		reader.ok = false
	}
	for i := 0; i < definitionCount && reader.ok; i++ {
		artifact.DefinitionFiles = append(artifact.DefinitionFiles, rtg.ImportSource{
			Filename: reader.string(), Source: append([]byte(nil), reader.bytes()...), Ok: true,
		})
	}
	enablement := reader.bytes()
	if len(enablement) != len(artifact.Enablement) {
		reader.ok = false
	} else {
		copy(artifact.Enablement[:], enablement)
	}
	libraryCount := reader.integer()
	if libraryCount < 0 || libraryCount > 4096 {
		reader.ok = false
	}
	for i := 0; i < libraryCount && reader.ok; i++ {
		artifact.LibraryFiles = append(artifact.LibraryFiles, rbe.File{
			Path: reader.string(), Source: append([]byte(nil), reader.bytes()...),
		})
	}
	if !reader.ok || reader.at != len(reader.source) || !validArtifactMetadata(artifact) {
		return Artifact{}, false
	}
	artifact.Payload = source[HeaderSize+metadataSize:]
	return artifact, true
}

func IsArtifact(source []byte) bool {
	return len(source) >= 4 && string(source[:4]) == Magic
}

func validArtifact(artifact Artifact) bool {
	return artifact.Generator > 0 && artifact.Generator <= 65535 &&
		artifact.Kernel > 0 && artifact.Kernel <= 65535 &&
		artifact.Protocol > 0 && artifact.Protocol <= 65535 &&
		artifact.Unit > 0 && artifact.Unit <= 65535 &&
		artifact.Optimization > 0 && artifact.Optimization <= 65535 &&
		len(artifact.Payload) > 0 && len(artifact.Payload) <= MaxPayload &&
		validArtifactMetadata(artifact)
}

func validArtifactMetadata(artifact Artifact) bool {
	descriptor := artifact.Descriptor
	return validLibraryFiles(artifact.LibraryFiles) &&
		descriptor.Name != "" && descriptor.Family != "" &&
		descriptor.OS != "" && descriptor.ISA != "" &&
		descriptor.Version > 0 && descriptor.Version <= 65535 &&
		descriptor.WordBits > 0 && descriptor.WordBits <= 65535 &&
		descriptor.PointerBits > 0 && descriptor.PointerBits <= 65535 &&
		descriptor.CodePointerBits > 0 && descriptor.CodePointerBits <= 65535 &&
		descriptor.FunctionPointerBits > 0 && descriptor.FunctionPointerBits <= 65535 &&
		descriptor.MaxAlign > 0 && descriptor.MaxAlign <= 65535 &&
		descriptor.ArenaDefault >= 0 &&
		(descriptor.Endian == "little" || descriptor.Endian == "big") &&
		descriptor.ABI != "" && descriptor.Runtime != "" &&
		descriptor.OutputKind != "" &&
		(descriptor.Executable != "" || descriptor.Object != "") &&
		artifact.Host != ""
}

func validLibraryFiles(files []rbe.File) bool {
	if len(files) > 4096 {
		return false
	}
	for i := 0; i < len(files); i++ {
		if !rbe.ValidLibraryPath(files[i].Path) || len(files[i].Source) > MaxMeta {
			return false
		}
		for j := 0; j < i; j++ {
			if files[j].Path == files[i].Path {
				return false
			}
		}
	}
	return true
}

func appendString(out []byte, value string) []byte {
	out = appendVarint(out, len(value))
	return append(out, value...)
}

func appendBytes(out []byte, value []byte) []byte {
	out = appendVarint(out, len(value))
	return append(out, value...)
}

func appendStrings(out []byte, values []string) []byte {
	out = appendVarint(out, len(values))
	for i := 0; i < len(values); i++ {
		out = appendString(out, values[i])
	}
	return out
}

func appendVarint(out []byte, value int) []byte {
	for value >= 128 {
		out = append(out, byte(value)|0x80)
		value >>= 7
	}
	return append(out, byte(value))
}

type metadataReader struct {
	source []byte
	at     int
	ok     bool
}

func (r *metadataReader) integer() int {
	if !r.ok {
		return 0
	}
	value := 0
	shift := 0
	for r.at < len(r.source) && shift <= 28 {
		next := r.source[r.at]
		r.at++
		value |= int(next&0x7f) << shift
		if next < 128 {
			return value
		}
		shift += 7
	}
	r.ok = false
	return 0
}

func (r *metadataReader) string() string {
	return string(r.bytes())
}

func (r *metadataReader) bytes() []byte {
	size := r.integer()
	if !r.ok || size < 0 || r.at+size < r.at || r.at+size > len(r.source) {
		r.ok = false
		return nil
	}
	value := r.source[r.at : r.at+size]
	r.at += size
	return value
}

func (r *metadataReader) strings() []string {
	count := r.integer()
	if !r.ok || count < 0 || count > 4096 {
		r.ok = false
		return nil
	}
	values := make([]string, count)
	for i := 0; i < count; i++ {
		values[i] = r.string()
	}
	return values
}

func put16(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
}

func put32(out []byte, at int, value int) {
	put16(out, at, value)
	put16(out, at+2, value>>16)
}

func read16(source []byte, at int) int {
	return int(source[at]) | int(source[at+1])<<8
}

func read32(source []byte, at int) int {
	return read16(source, at) | read16(source, at+2)<<16
}

func checksum(source []byte) (int, int) {
	a, b := 1, 0
	for i := 0; i < len(source); i++ {
		a = (a + int(source[i])) % 65521
		b = (b + a) % 65521
	}
	return a, b
}
