// Package rfeabi defines the GC-visible context shared by the emulator and
// native-call runtime. Native byte offsets are declared by the emitter and
// verified against these types before any entry is admitted.
package rfeabi

type Page struct {
	Number      uint64
	Data        *[4096]byte
	Permissions uint64
	Epoch       *uint64
}

type Descriptor struct {
	PC, Entry, Instructions, Reserved uint64
	Prefix                            [17]uint8
	Padding                           [15]uint8
}

// Context contains no owner token. Encoded borrowed addresses must be zero
// outside the serialized native call. Pointers remain typed for Go's GC;
// foreign execution uses a separately pinned, pointer-free staging image.
type Context struct {
	Retired, Status, Address                   uint64
	Clock                                      *uint64
	Pages                                      [64]Page
	Remaining, Total, MemoryTotal              uint64
	CodeView                                   [4]uint64
	Blocks                                     [1024]Descriptor
	LoopExits, LoopIterations, PreparedTargets uint64
	DescriptorBase, AdmissionEpoch             uint64
	// Owned direct window; usable only by registered foreign fault sites.
	DirectGuest, DirectHost, DirectSize uint64
}
