// Package resultabi implements the target-independent memory protocol used by
// the compact backend omnibus.
package resultabi

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Block is the exact byte representation exported by a target program.
// Fields are always little-endian, independently of the target byte order.
type Block [Size]byte

// Snapshot is a decoded, internally consistent observation of a Block.
type Snapshot struct {
	Profile         uint32 `json:"profile"`
	State           State  `json:"state"`
	CurrentProbe    uint32 `json:"current_probe"`
	FailureProbe    uint32 `json:"failure_probe"`
	CompletedProbes uint32 `json:"completed_probes"`
	Flags           uint32 `json:"flags"`
	Expected        uint64 `json:"expected"`
	Observed        uint64 `json:"observed"`
	Signature       uint64 `json:"signature"`
	Sequence        uint32 `json:"sequence"`
}

// New returns a fully initialized, not-started result block.
func New(profile uint32) Block {
	var block Block
	block.Reset(profile)
	return block
}

// Reset initializes the protocol header. State is committed last.
func (b *Block) Reset(profile uint32) {
	for i := range b {
		b[i] = 0
	}
	put32(b[:], OffsetMagic, Magic)
	put16(b[:], OffsetVersion, Version)
	put16(b[:], OffsetSize, Size)
	put32(b[:], OffsetProfile, profile)
	b.setState(StateNotStarted)
}

// BeginProbe publishes the probe identifier before publishing Running.
func (b *Block) BeginProbe(id uint32) {
	put32(b[:], OffsetCurrentProbe, id)
	b.bumpSequence()
	b.setState(StateRunning)
}

// CompleteProbe records a completed probe and advances the deterministic
// signature. The block remains Running until Pass commits the terminal state.
func (b *Block) CompleteProbe(id uint32, observed uint64) {
	put32(b[:], OffsetCurrentProbe, id)
	completed := get32(b[:], OffsetCompletedProbes) + 1
	put32(b[:], OffsetCompletedProbes, completed)
	signature := MixSignature(get64(b[:], OffsetSignature), id, observed)
	put64(b[:], OffsetSignature, signature)
	b.bumpSequence()
}

// FailComparison preserves all diagnostic fields before committing failure.
func (b *Block) FailComparison(id uint32, expected uint64, observed uint64) {
	put32(b[:], OffsetCurrentProbe, id)
	put32(b[:], OffsetFailureProbe, id)
	put64(b[:], OffsetExpected, expected)
	put64(b[:], OffsetObserved, observed)
	b.bumpSequence()
	b.setState(StateFailedComparison)
}

// MarkTrap records a trap/reset observation when a target adapter can do so.
// The current probe is deliberately retained.
func (b *Block) MarkTrap() {
	b.bumpSequence()
	b.setState(StateTrapReset)
}

// MarkProfileMismatch records the expected and observed profile identifiers.
func (b *Block) MarkProfileMismatch(expected uint32, observed uint32) {
	put64(b[:], OffsetExpected, uint64(expected))
	put64(b[:], OffsetObserved, uint64(observed))
	b.bumpSequence()
	b.setState(StateProfileMismatch)
}

// Pass commits the final signature and then the terminal state.
func (b *Block) Pass(signature uint64) {
	put64(b[:], OffsetSignature, signature)
	b.bumpSequence()
	b.setState(StatePassed)
}

// Decode validates the fixed protocol header and returns its payload.
func Decode(data []byte) (Snapshot, error) {
	if len(data) < Size {
		return Snapshot{}, fmt.Errorf("result block is %d bytes; need %d", len(data), Size)
	}
	if got := get32(data, OffsetMagic); got != Magic {
		return Snapshot{}, fmt.Errorf("result magic %#x; want %#x", got, Magic)
	}
	if got := get16(data, OffsetVersion); got != Version {
		return Snapshot{}, fmt.Errorf("result version %d; want %d", got, Version)
	}
	if got := get16(data, OffsetSize); got != Size {
		return Snapshot{}, fmt.Errorf("result size field %d; want %d", got, Size)
	}
	state := State(get32(data, OffsetState))
	if !state.Valid() {
		return Snapshot{}, fmt.Errorf("unknown result state %d", state)
	}
	return Snapshot{
		Profile:         get32(data, OffsetProfile),
		State:           state,
		CurrentProbe:    get32(data, OffsetCurrentProbe),
		FailureProbe:    get32(data, OffsetFailureProbe),
		CompletedProbes: get32(data, OffsetCompletedProbes),
		Flags:           get32(data, OffsetFlags),
		Expected:        get64(data, OffsetExpected),
		Observed:        get64(data, OffsetObserved),
		Signature:       get64(data, OffsetSignature),
		Sequence:        get32(data, OffsetSequence),
	}, nil
}

// ValidatePass requires the terminal state, profile, and independently known
// signature to agree. A passed flag by itself is never sufficient.
func (s Snapshot) ValidatePass(profile uint32, signature uint64) error {
	if s.Profile != profile {
		return fmt.Errorf("result profile %#x; want %#x", s.Profile, profile)
	}
	if s.State != StatePassed {
		if s.State == StateFailedComparison {
			return fmt.Errorf("probe %d failed: expected %#x, observed %#x", s.FailureProbe, s.Expected, s.Observed)
		}
		return fmt.Errorf("result state is %s at probe %d", s.State, s.CurrentProbe)
	}
	if s.Signature != signature {
		return fmt.Errorf("result signature %#x; want %#x", s.Signature, signature)
	}
	return nil
}

// MixSignature uses only defined unsigned 32-bit arithmetic in each half so
// the same operation is cheap to reproduce in C89 on 32-bit targets.
func MixSignature(signature uint64, probeID uint32, observed uint64) uint64 {
	lo := uint32(signature)
	hi := uint32(signature >> 32)
	lo = (lo ^ probeID ^ uint32(observed)) * uint32(16777619)
	hi = (hi ^ probeID ^ uint32(observed>>32)) * uint32(2246822519)
	return uint64(hi)<<32 | uint64(lo)
}

func (s State) Valid() bool {
	return s >= StateNotStarted && s <= StateProfileMismatch
}

func (s State) String() string {
	switch s {
	case StateNotStarted:
		return "not_started"
	case StateRunning:
		return "running"
	case StatePassed:
		return "passed"
	case StateFailedComparison:
		return "failed_comparison"
	case StateTrapReset:
		return "trap_reset"
	case StateProfileMismatch:
		return "profile_mismatch"
	default:
		return fmt.Sprintf("state(%d)", s)
	}
}

func (b *Block) bumpSequence() {
	put32(b[:], OffsetSequence, get32(b[:], OffsetSequence)+1)
}

func (b *Block) setState(state State) {
	put32(b[:], OffsetState, uint32(state))
}

func get16(data []byte, offset int) uint16 {
	return binary.LittleEndian.Uint16(data[offset : offset+2])
}

func get32(data []byte, offset int) uint32 {
	return binary.LittleEndian.Uint32(data[offset : offset+4])
}

func get64(data []byte, offset int) uint64 {
	return binary.LittleEndian.Uint64(data[offset : offset+8])
}

func put16(data []byte, offset int, value uint16) {
	binary.LittleEndian.PutUint16(data[offset:offset+2], value)
}

func put32(data []byte, offset int, value uint32) {
	binary.LittleEndian.PutUint32(data[offset:offset+4], value)
}

func put64(data []byte, offset int, value uint64) {
	binary.LittleEndian.PutUint64(data[offset:offset+8], value)
}

var errAddressBeforeBase = errors.New("result symbol address is below memory dump base")
