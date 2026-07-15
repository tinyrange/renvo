package target

import "fmt"

// ExecutionModel distinguishes objects that require an operating-system
// process from objects that can be composed with a freestanding board.
type ExecutionModel string

const (
	ExecutionHosted       ExecutionModel = "hosted"
	ExecutionFreestanding ExecutionModel = "freestanding"
)

type Endian string

const (
	EndianLittle Endian = "little"
	EndianBig    Endian = "big"
)

// ObjectFormat describes the target-visible properties that both the native
// and C-reference emitters must agree on. FlagsMask selects ABI-significant
// object-header flags; other flags may describe optional ISA use.
type ObjectFormat struct {
	Name        string
	Container   string
	AddressBits int
	Endian      Endian
	MachineID   uint16
	FlagsMask   uint32
	FlagsValue  uint32
}

// ObjectTarget is reusable without a board. An emitter consumes this contract
// to produce relocatable objects; firmware composition supplies a Board later.
type ObjectTarget struct {
	Name        string
	Execution   ExecutionModel
	ISA         string
	ABI         string
	IntBits     int
	PointerBits int
	Format      ObjectFormat
}

func (t ObjectTarget) Validate() error {
	if t.Name == "" || t.ISA == "" || t.ABI == "" {
		return fmt.Errorf("object target name, ISA, and ABI are required")
	}
	if t.Execution != ExecutionHosted && t.Execution != ExecutionFreestanding {
		return fmt.Errorf("object target %q has invalid execution model %q", t.Name, t.Execution)
	}
	if !supportedMachineWidth(t.IntBits) || !supportedMachineWidth(t.PointerBits) {
		return fmt.Errorf("object target %q requires 16-, 32-, or 64-bit int and pointer widths", t.Name)
	}
	if t.Format.Name == "" || t.Format.Container == "" || t.Format.MachineID == 0 {
		return fmt.Errorf("object target %q requires an object format name, container, and machine ID", t.Name)
	}
	if t.Format.AddressBits != 32 && t.Format.AddressBits != 64 {
		return fmt.Errorf("object target %q has unsupported object address width %d", t.Name, t.Format.AddressBits)
	}
	if t.Format.Endian != EndianLittle && t.Format.Endian != EndianBig {
		return fmt.Errorf("object target %q has invalid object byte order %q", t.Name, t.Format.Endian)
	}
	if t.Format.FlagsValue&^t.Format.FlagsMask != 0 {
		return fmt.Errorf("object target %q has ABI flag values outside its mask", t.Name)
	}
	return nil
}

func supportedMachineWidth(bits int) bool {
	return bits == 16 || bits == 32 || bits == 64
}

// ExpectedArtifactFormat is useful for constructing synthetic linked-artifact
// descriptions. Real artifacts obtain the same fields from their file header.
func (t ObjectTarget) ExpectedArtifactFormat() ArtifactFormat {
	return ArtifactFormat{
		Container:   t.Format.Container,
		AddressBits: t.Format.AddressBits,
		Endian:      t.Format.Endian,
		MachineID:   t.Format.MachineID,
		Flags:       t.Format.FlagsValue,
	}
}

// C89Profile derives the C reference backend's static machine assumptions
// from the same object contract used by native emitters.
func (t ObjectTarget) C89Profile(runtimeOps ...string) (CMachineProfile, error) {
	if err := t.Validate(); err != nil {
		return CMachineProfile{}, err
	}
	endian := CEndianLittle
	if t.Format.Endian == EndianBig {
		endian = CEndianBig
	}
	profile := C89ExplicitProfile(t.Name, t.Execution == ExecutionHosted, t.IntBits, t.PointerBits, endian, t.ABI, runtimeOps...)
	if err := profile.Validate(); err != nil {
		return CMachineProfile{}, err
	}
	return profile, nil
}

type BSSInitialization string

const (
	BSSZeroedByStartup BSSInitialization = "zero-by-startup"
)

type StackDirection string

const (
	StackGrowsDown StackDirection = "down"
	StackGrowsUp   StackDirection = "up"
)

type HeapModel string

const (
	HeapNone     HeapModel = "none"
	HeapBump     HeapModel = "bump"
	HeapExternal HeapModel = "external"
)

type OOMPolicy string

const (
	OOMNone   OOMPolicy = "none"
	OOMTrap   OOMPolicy = "trap"
	OOMResult OOMPolicy = "result"
	OOMPanic  OOMPolicy = "panic"
)

type InterruptModel string

const (
	InterruptNone     InterruptModel = "none"
	InterruptVectored InterruptModel = "vectored"
)

type VolatileAlignment string

const (
	VolatileByteAligned    VolatileAlignment = "byte"
	VolatileNaturalAligned VolatileAlignment = "natural"
)

type ResultTransport string

const (
	ResultTransportNone           ResultTransport = "none"
	ResultTransportDebuggerMemory ResultTransport = "debugger-memory"
)

type VolatileWidths uint8

const (
	VolatileWidth8 VolatileWidths = 1 << iota
	VolatileWidth16
	VolatileWidth32
	VolatileWidth64
)

const allVolatileWidths = VolatileWidth8 | VolatileWidth16 | VolatileWidth32 | VolatileWidth64

type StartupContract struct {
	VectorSymbol    string
	VectorAddress   uint64
	VectorAlignment uint64
	EntryAlignment  uint64
	BSS             BSSInitialization
}

type StackContract struct {
	InitialPointer uint64
	DefaultSize    uint64
	GuardSize      uint64
	Alignment      uint64
	Direction      StackDirection
}

type HeapContract struct {
	Model HeapModel
	OOM   OOMPolicy
}

type InterruptContract struct {
	Model InterruptModel
}

type VolatileContract struct {
	Widths    VolatileWidths
	Alignment VolatileAlignment
}

type ResultContract struct {
	Transport ResultTransport
	Symbol    string
}

// RuntimeContract lists only operations and unresolved imports supplied by
// board/runtime composition. Nothing hosted is inherited implicitly.
type RuntimeContract struct {
	Operations      []string
	ProvidedImports []string
	Heap            HeapContract
	Interrupts      InterruptContract
	Volatile        VolatileContract
	Result          ResultContract
}

type Board struct {
	Name    string
	Regions []Region
	Startup StartupContract
	Stack   StackContract
	Runtime RuntimeContract
}

// Composition combines a reusable relocatable-object target with the board
// contracts required to construct and validate a flashable firmware image.
type Composition struct {
	Object ObjectTarget
	Board  Board
}

func (c Composition) Validate() error {
	_, _, violations := validateComposition(c)
	if len(violations) != 0 {
		return violations[0]
	}
	return nil
}

func validateCompositionContract(c Composition) error {
	if err := c.Object.Validate(); err != nil {
		return err
	}
	if c.Object.Execution != ExecutionFreestanding {
		return fmt.Errorf("board %q requires a freestanding object target", c.Board.Name)
	}
	if err := validateBoardContract(c.Board); err != nil {
		return err
	}
	return nil
}

// C89Profile derives the reference-backend contract including only runtime
// operations explicitly supplied by this composition.
func (c Composition) C89Profile() (CMachineProfile, error) {
	if err := c.Validate(); err != nil {
		return CMachineProfile{}, err
	}
	return c.Object.C89Profile(c.Board.Runtime.Operations...)
}

func validateBoardContract(board Board) error {
	if board.Name == "" {
		return fmt.Errorf("board name is required")
	}
	if board.Startup.VectorSymbol == "" {
		return fmt.Errorf("board %q requires a vector symbol", board.Name)
	}
	if board.Startup.VectorAlignment == 0 || !powerOfTwo(board.Startup.VectorAlignment) || board.Startup.EntryAlignment == 0 || !powerOfTwo(board.Startup.EntryAlignment) {
		return fmt.Errorf("board %q requires power-of-two vector and entry alignment", board.Name)
	}
	if board.Startup.VectorAddress%board.Startup.VectorAlignment != 0 {
		return fmt.Errorf("board %q vector address is not aligned", board.Name)
	}
	if board.Startup.BSS != BSSZeroedByStartup {
		return fmt.Errorf("board %q must explicitly define BSS initialization", board.Name)
	}
	stack := board.Stack
	if stack.InitialPointer == 0 || stack.DefaultSize == 0 || stack.Alignment == 0 || !powerOfTwo(stack.Alignment) {
		return fmt.Errorf("board %q requires stack pointer, size, and power-of-two alignment", board.Name)
	}
	if stack.Direction != StackGrowsDown && stack.Direction != StackGrowsUp {
		return fmt.Errorf("board %q has invalid stack direction %q", board.Name, stack.Direction)
	}
	if stack.InitialPointer%stack.Alignment != 0 || stack.DefaultSize%stack.Alignment != 0 || stack.GuardSize%stack.Alignment != 0 {
		return fmt.Errorf("board %q stack pointer, size, and guard must satisfy stack alignment", board.Name)
	}
	if err := validateRuntimeContract(board.Name, board.Runtime); err != nil {
		return err
	}
	return nil
}

func validateRuntimeContract(boardName string, runtime RuntimeContract) error {
	if runtime.Heap.Model != HeapNone && runtime.Heap.Model != HeapBump && runtime.Heap.Model != HeapExternal {
		return fmt.Errorf("board %q has invalid heap model %q", boardName, runtime.Heap.Model)
	}
	if runtime.Heap.Model == HeapNone {
		if runtime.Heap.OOM != OOMNone {
			return fmt.Errorf("board %q has an OOM policy without a heap", boardName)
		}
	} else if runtime.Heap.OOM != OOMTrap && runtime.Heap.OOM != OOMResult && runtime.Heap.OOM != OOMPanic {
		return fmt.Errorf("board %q heap requires an explicit OOM policy", boardName)
	}
	if runtime.Interrupts.Model != InterruptNone && runtime.Interrupts.Model != InterruptVectored {
		return fmt.Errorf("board %q has invalid interrupt model %q", boardName, runtime.Interrupts.Model)
	}
	if runtime.Volatile.Widths == 0 || runtime.Volatile.Widths&^allVolatileWidths != 0 {
		return fmt.Errorf("board %q requires supported volatile access widths", boardName)
	}
	if runtime.Volatile.Alignment != VolatileByteAligned && runtime.Volatile.Alignment != VolatileNaturalAligned {
		return fmt.Errorf("board %q requires a volatile alignment contract", boardName)
	}
	if runtime.Result.Transport != ResultTransportNone && runtime.Result.Transport != ResultTransportDebuggerMemory {
		return fmt.Errorf("board %q has invalid result transport %q", boardName, runtime.Result.Transport)
	}
	if runtime.Result.Transport == ResultTransportDebuggerMemory && runtime.Result.Symbol == "" {
		return fmt.Errorf("board %q debugger-memory result transport requires a symbol", boardName)
	}
	if runtime.Result.Transport == ResultTransportNone && runtime.Result.Symbol != "" {
		return fmt.Errorf("board %q has a result symbol without a transport", boardName)
	}
	if err := validateUniqueContractNames("runtime operation", runtime.Operations); err != nil {
		return fmt.Errorf("board %q: %w", boardName, err)
	}
	if err := validateUniqueStrings("provided import", runtime.ProvidedImports); err != nil {
		return fmt.Errorf("board %q: %w", boardName, err)
	}
	if err := validateRuntimeOperation(runtime.Operations, "heap", runtime.Heap.Model != HeapNone); err != nil {
		return fmt.Errorf("board %q: %w", boardName, err)
	}
	if err := validateRuntimeOperation(runtime.Operations, "interrupts", runtime.Interrupts.Model != InterruptNone); err != nil {
		return fmt.Errorf("board %q: %w", boardName, err)
	}
	if err := validateRuntimeOperation(runtime.Operations, "volatile_memory", runtime.Volatile.Widths != 0); err != nil {
		return fmt.Errorf("board %q: %w", boardName, err)
	}
	if err := validateRuntimeOperation(runtime.Operations, "result", runtime.Result.Transport != ResultTransportNone); err != nil {
		return fmt.Errorf("board %q: %w", boardName, err)
	}
	return nil
}

func validateUniqueContractNames(kind string, values []string) error {
	for i, value := range values {
		if !validCProfileName(value) {
			return fmt.Errorf("invalid %s %q", kind, value)
		}
		for j := 0; j < i; j++ {
			if values[j] == value {
				return fmt.Errorf("duplicate %s %q", kind, value)
			}
		}
	}
	return nil
}

func validateUniqueStrings(kind string, values []string) error {
	for i, value := range values {
		if value == "" {
			return fmt.Errorf("empty %s", kind)
		}
		for j := 0; j < i; j++ {
			if values[j] == value {
				return fmt.Errorf("duplicate %s %q", kind, value)
			}
		}
	}
	return nil
}

func validateRuntimeOperation(operations []string, name string, supplied bool) error {
	advertised := stringInList(operations, name)
	if advertised && !supplied {
		return fmt.Errorf("runtime operation %q is advertised without a contract", name)
	}
	if supplied && !advertised {
		return fmt.Errorf("runtime contract %q is not advertised", name)
	}
	return nil
}

func powerOfTwo(value uint64) bool {
	return value != 0 && value&(value-1) == 0
}
