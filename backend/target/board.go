// Package target defines host-side composition and resource checks for
// freestanding RENVO artifacts. Instruction selection and board composition are
// intentionally separate contracts.
package target

import "fmt"

type RegionKind int

const (
	RegionFlash RegionKind = iota + 1
	RegionRAM
	RegionReserved
)

type Region struct {
	Name  string
	Kind  RegionKind
	Start uint64
	Size  uint64
}

func (r Region) End() uint64 {
	return safeEnd(r.Start, r.Size)
}

type SectionFlags uint32

const (
	SectionAlloc SectionFlags = 1 << iota
	SectionWrite
	SectionExec
)

type Section struct {
	Name        string
	Address     uint64
	Size        uint64
	LoadAddress uint64
	LoadSize    uint64
	Flags       SectionFlags
}

// ArtifactFormat is read from the final linked image. It is deliberately
// separate from ObjectFormat: the latter is an expectation, while this is
// evidence supplied by the artifact being validated.
type ArtifactFormat struct {
	Container   string
	AddressBits int
	Endian      Endian
	MachineID   uint16
	Flags       uint32
}

type Artifact struct {
	Format        ArtifactFormat
	Entry         uint64
	VectorSymbol  string
	VectorAddress uint64
	Sections      []Section
	Imports       []string
	HeapSize      uint64
	StackSize     uint64
}

type Usage struct {
	FlashUsed     uint64
	FlashCapacity uint64
	FlashFree     uint64
	RAMStatic     uint64
	HeapReserved  uint64
	StackReserved uint64
	GuardReserved uint64
	RAMCapacity   uint64
	RAMFree       uint64
}

type ViolationCode string

const (
	ViolationBoard            ViolationCode = "invalid-board"
	ViolationObjectTarget     ViolationCode = "object-target-mismatch"
	ViolationSectionRegion    ViolationCode = "section-outside-region"
	ViolationSectionOverlap   ViolationCode = "section-overlap"
	ViolationReservedOverlap  ViolationCode = "reserved-overlap"
	ViolationLoadRegion       ViolationCode = "load-outside-flash"
	ViolationFlashBudget      ViolationCode = "flash-budget"
	ViolationRAMBudget        ViolationCode = "ram-budget"
	ViolationStackOverlap     ViolationCode = "stack-overlap"
	ViolationEntry            ViolationCode = "bad-entry"
	ViolationVector           ViolationCode = "bad-vector"
	ViolationRuntime          ViolationCode = "unsupported-runtime"
	ViolationUnresolvedImport ViolationCode = "forbidden-import"
)

type Violation struct {
	Code    ViolationCode
	Subject string
	Detail  string
}

func (v Violation) Error() string {
	if v.Subject == "" {
		return fmt.Sprintf("%s: %s", v.Code, v.Detail)
	}
	return fmt.Sprintf("%s %s: %s", v.Code, v.Subject, v.Detail)
}

type Validation struct {
	Usage      Usage
	Violations []Violation
}

func (v Validation) OK() bool {
	return len(v.Violations) == 0
}

func Validate(composition Composition, artifact Artifact) Validation {
	var result Validation
	board := composition.Board
	flash, ram, boardViolations := validateComposition(composition)
	result.Violations = append(result.Violations, boardViolations...)
	if !artifactMatchesObjectTarget(composition.Object, artifact.Format) {
		result.Violations = append(result.Violations, Violation{
			Code: ViolationObjectTarget,
			Detail: fmt.Sprintf("artifact is %s%d/%s machine %#x flags %#x; want %s%d/%s machine %#x flags %#x/%#x",
				artifact.Format.Container, artifact.Format.AddressBits, artifact.Format.Endian, artifact.Format.MachineID, artifact.Format.Flags,
				composition.Object.Format.Container, composition.Object.Format.AddressBits, composition.Object.Format.Endian,
				composition.Object.Format.MachineID, composition.Object.Format.FlagsValue, composition.Object.Format.FlagsMask),
		})
	}
	var flashCapacityOverflow bool
	var ramCapacityOverflow bool
	result.Usage.FlashCapacity, flashCapacityOverflow = usableRegionCapacity(flash, board.Regions)
	result.Usage.RAMCapacity, ramCapacityOverflow = usableRegionCapacity(ram, board.Regions)
	if flashCapacityOverflow || ramCapacityOverflow {
		result.Violations = append(result.Violations, Violation{Code: ViolationBoard, Detail: "aggregate memory-region capacity overflows the address space"})
	}

	stackSize := artifact.StackSize
	if stackSize == 0 {
		stackSize = board.Stack.DefaultSize
	}
	result.Usage.HeapReserved = artifact.HeapSize
	if artifact.HeapSize != 0 && board.Runtime.Heap.Model == HeapNone {
		result.Violations = append(result.Violations, Violation{Code: ViolationRuntime, Subject: "heap", Detail: "artifact reserves a heap that the board runtime does not supply"})
	}
	result.Usage.StackReserved = stackSize
	result.Usage.GuardReserved = board.Stack.GuardSize

	stackStart, stackReservation, stackValid := stackRange(board.Stack, stackSize)
	if !aligned(stackSize, board.Stack.Alignment) {
		stackValid = false
		result.Violations = append(result.Violations, Violation{Code: ViolationStackOverlap, Detail: fmt.Sprintf("stack size %d does not satisfy %d-byte alignment", stackSize, board.Stack.Alignment)})
	}
	if stackValid && !containedByRegions(ram, stackStart, stackReservation) {
		stackValid = false
		result.Violations = append(result.Violations, Violation{Code: ViolationStackOverlap, Detail: "configured stack and guard are not contained by one RAM region"})
	}
	if stackValid {
		for _, region := range board.Regions {
			if region.Kind == RegionReserved && rangesOverlap(stackStart, stackReservation, region.Start, region.Size) {
				result.Violations = append(result.Violations, Violation{Code: ViolationStackOverlap, Subject: region.Name, Detail: "reserved region overlaps configured stack or guard"})
			}
		}
	}
	flashUsageOverflow := false
	ramUsageOverflow := false

	for i := range artifact.Sections {
		section := artifact.Sections[i]
		if section.Flags&SectionAlloc == 0 || section.Size == 0 {
			continue
		}
		want := RegionFlash
		if section.Flags&SectionWrite != 0 {
			want = RegionRAM
		}
		if !containedByKind(board.Regions, want, section.Address, section.Size) {
			result.Violations = append(result.Violations, Violation{Code: ViolationSectionRegion, Subject: section.Name, Detail: fmt.Sprintf("%#x..%#x is not in %s", section.Address, safeEnd(section.Address, section.Size), regionKindName(want))})
		}
		if want == RegionFlash {
			result.Usage.FlashUsed, flashUsageOverflow = addWithOverflow(result.Usage.FlashUsed, section.Size, flashUsageOverflow)
		} else {
			result.Usage.RAMStatic, ramUsageOverflow = addWithOverflow(result.Usage.RAMStatic, section.Size, ramUsageOverflow)
			if stackValid && rangesOverlap(section.Address, section.Size, stackStart, stackReservation) {
				result.Violations = append(result.Violations, Violation{Code: ViolationStackOverlap, Subject: section.Name, Detail: "section overlaps configured stack or guard"})
			}
		}
		for _, reserved := range board.Regions {
			if reserved.Kind == RegionReserved && rangesOverlap(section.Address, section.Size, reserved.Start, reserved.Size) {
				result.Violations = append(result.Violations, Violation{Code: ViolationReservedOverlap, Subject: section.Name, Detail: "overlaps " + reserved.Name})
			}
		}
		if section.LoadSize > 0 {
			if !containedByRegions(flash, section.LoadAddress, section.LoadSize) {
				result.Violations = append(result.Violations, Violation{Code: ViolationLoadRegion, Subject: section.Name, Detail: "load image is not in flash"})
			}
			if !(want == RegionFlash && section.LoadAddress == section.Address && section.LoadSize == section.Size) {
				result.Usage.FlashUsed, flashUsageOverflow = addWithOverflow(result.Usage.FlashUsed, section.LoadSize, flashUsageOverflow)
			}
		}
		for j := 0; j < i; j++ {
			other := artifact.Sections[j]
			if other.Flags&SectionAlloc != 0 && rangesOverlap(section.Address, section.Size, other.Address, other.Size) {
				result.Violations = append(result.Violations, Violation{Code: ViolationSectionOverlap, Subject: section.Name, Detail: "overlaps " + other.Name})
			}
		}
	}

	if flashUsageOverflow || result.Usage.FlashUsed > result.Usage.FlashCapacity {
		result.Violations = append(result.Violations, Violation{Code: ViolationFlashBudget, Detail: fmt.Sprintf("uses %d bytes; capacity is %d", result.Usage.FlashUsed, result.Usage.FlashCapacity)})
	} else {
		result.Usage.FlashFree = result.Usage.FlashCapacity - result.Usage.FlashUsed
	}
	ramNeed, ramNeedOverflow := saturatingAdd(result.Usage.RAMStatic, result.Usage.HeapReserved)
	ramNeed, ramNeedOverflow = addWithOverflow(ramNeed, result.Usage.StackReserved, ramNeedOverflow)
	ramNeed, ramNeedOverflow = addWithOverflow(ramNeed, result.Usage.GuardReserved, ramNeedOverflow)
	if ramUsageOverflow || ramNeedOverflow || ramNeed > result.Usage.RAMCapacity {
		result.Violations = append(result.Violations, Violation{Code: ViolationRAMBudget, Detail: fmt.Sprintf("needs %d bytes; capacity is %d", ramNeed, result.Usage.RAMCapacity)})
	} else {
		result.Usage.RAMFree = result.Usage.RAMCapacity - ramNeed
	}

	if artifact.VectorSymbol != board.Startup.VectorSymbol || artifact.VectorAddress != board.Startup.VectorAddress || !aligned(artifact.VectorAddress, board.Startup.VectorAlignment) || !addressInAllocatedSection(artifact.Sections, artifact.VectorAddress, SectionExec) {
		result.Violations = append(result.Violations, Violation{Code: ViolationVector, Detail: fmt.Sprintf("vector %q at %#x; want %q at %#x with %d-byte alignment in an executable section", artifact.VectorSymbol, artifact.VectorAddress, board.Startup.VectorSymbol, board.Startup.VectorAddress, board.Startup.VectorAlignment)})
	}
	if !aligned(artifact.Entry, board.Startup.EntryAlignment) || !addressInAllocatedSection(artifact.Sections, artifact.Entry, SectionExec) {
		result.Violations = append(result.Violations, Violation{Code: ViolationEntry, Detail: fmt.Sprintf("entry %#x is not %d-byte aligned in an executable section", artifact.Entry, board.Startup.EntryAlignment)})
	}
	for _, name := range artifact.Imports {
		if !stringInList(board.Runtime.ProvidedImports, name) {
			result.Violations = append(result.Violations, Violation{Code: ViolationUnresolvedImport, Subject: name, Detail: "not provided by the freestanding runtime contract"})
		}
	}
	return result
}

func validateComposition(composition Composition) ([]Region, []Region, []Violation) {
	board := composition.Board
	var flash []Region
	var ram []Region
	var violations []Violation
	if err := validateCompositionContract(composition); err != nil {
		violations = append(violations, Violation{Code: ViolationBoard, Detail: err.Error()})
	}
	for i, region := range board.Regions {
		if _, valid := checkedEnd(region.Start, region.Size); region.Name == "" || !valid {
			violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "region has an invalid name or range"})
		}
		if region.Kind == RegionFlash {
			flash = append(flash, region)
		} else if region.Kind == RegionRAM {
			ram = append(ram, region)
		} else if region.Kind != RegionReserved {
			violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "unknown region kind"})
		}
		for j := 0; j < i; j++ {
			other := board.Regions[j]
			if region.Kind != RegionReserved && other.Kind != RegionReserved && rangesOverlap(region.Start, region.Size, other.Start, other.Size) {
				violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "overlaps board region " + other.Name})
			}
			if region.Kind == RegionReserved && other.Kind == RegionReserved && rangesOverlap(region.Start, region.Size, other.Start, other.Size) {
				violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "overlaps reserved region " + other.Name})
			}
		}
	}
	if len(flash) == 0 || len(ram) == 0 {
		violations = append(violations, Violation{Code: ViolationBoard, Detail: "at least one flash and one RAM region are required"})
	}
	if start, size, ok := stackRange(board.Stack, board.Stack.DefaultSize); !ok || !containedByRegions(ram, start, size) {
		violations = append(violations, Violation{Code: ViolationBoard, Detail: "default stack and guard are not contained by one RAM region"})
	} else {
		for _, region := range board.Regions {
			if region.Kind == RegionReserved && rangesOverlap(start, size, region.Start, region.Size) {
				violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "reserved region overlaps default stack or guard"})
			}
		}
	}
	return flash, ram, violations
}

func artifactMatchesObjectTarget(target ObjectTarget, artifact ArtifactFormat) bool {
	format := target.Format
	return artifact.Container == format.Container &&
		artifact.AddressBits == format.AddressBits &&
		artifact.Endian == format.Endian &&
		artifact.MachineID == format.MachineID &&
		artifact.Flags&format.FlagsMask == format.FlagsValue
}

func stackRange(stack StackContract, stackSize uint64) (uint64, uint64, bool) {
	reservation, overflow := saturatingAdd(stackSize, stack.GuardSize)
	if overflow || reservation == 0 {
		return 0, reservation, false
	}
	if stack.Direction == StackGrowsDown {
		if reservation > stack.InitialPointer {
			return 0, reservation, false
		}
		return stack.InitialPointer - reservation, reservation, true
	}
	if stack.Direction == StackGrowsUp {
		if _, valid := checkedEnd(stack.InitialPointer, reservation); !valid {
			return 0, reservation, false
		}
		return stack.InitialPointer, reservation, true
	}
	return 0, reservation, false
}

func aligned(value uint64, alignment uint64) bool {
	return alignment != 0 && value%alignment == 0
}

func usableRegionCapacity(regions []Region, memoryMap []Region) (uint64, bool) {
	var total uint64
	overflow := false
	for _, region := range regions {
		usable := region.Size
		for _, reserved := range memoryMap {
			if reserved.Kind != RegionReserved {
				continue
			}
			overlap, valid := rangeIntersectionSize(region.Start, region.Size, reserved.Start, reserved.Size)
			if !valid || overlap > usable {
				overflow = true
				usable = 0
				continue
			}
			usable -= overlap
		}
		total, overflow = addWithOverflow(total, usable, overflow)
	}
	return total, overflow
}

func rangeIntersectionSize(leftStart uint64, leftSize uint64, rightStart uint64, rightSize uint64) (uint64, bool) {
	leftEnd, leftValid := checkedEnd(leftStart, leftSize)
	rightEnd, rightValid := checkedEnd(rightStart, rightSize)
	if !leftValid || !rightValid {
		return 0, false
	}
	start := leftStart
	if rightStart > start {
		start = rightStart
	}
	end := leftEnd
	if rightEnd < end {
		end = rightEnd
	}
	if start >= end {
		return 0, true
	}
	return end - start, true
}

func containedByKind(regions []Region, kind RegionKind, start uint64, size uint64) bool {
	for _, region := range regions {
		if region.Kind == kind && rangeContains(region.Start, region.Size, start, size) {
			return true
		}
	}
	return false
}

func containedByRegions(regions []Region, start uint64, size uint64) bool {
	for _, region := range regions {
		if rangeContains(region.Start, region.Size, start, size) {
			return true
		}
	}
	return false
}

func rangeContains(outerStart uint64, outerSize uint64, innerStart uint64, innerSize uint64) bool {
	outerEnd, outerValid := checkedEnd(outerStart, outerSize)
	innerEnd, innerValid := checkedEnd(innerStart, innerSize)
	return outerValid && innerValid && innerStart >= outerStart && innerEnd <= outerEnd
}

func rangesOverlap(leftStart uint64, leftSize uint64, rightStart uint64, rightSize uint64) bool {
	leftEnd, leftValid := checkedEnd(leftStart, leftSize)
	rightEnd, rightValid := checkedEnd(rightStart, rightSize)
	return leftValid && rightValid && leftStart < rightEnd && rightStart < leftEnd
}

func safeEnd(start uint64, size uint64) uint64 {
	if end, valid := checkedEnd(start, size); valid {
		return end
	}
	return ^uint64(0)
}

func checkedEnd(start uint64, size uint64) (uint64, bool) {
	if size == 0 || start > ^uint64(0)-size {
		return 0, false
	}
	return start + size, true
}

func saturatingAdd(left uint64, right uint64) (uint64, bool) {
	if left > ^uint64(0)-right {
		return ^uint64(0), true
	}
	return left + right, false
}

func addWithOverflow(left uint64, right uint64, alreadyOverflowed bool) (uint64, bool) {
	if alreadyOverflowed {
		return ^uint64(0), true
	}
	return saturatingAdd(left, right)
}

func addressInAllocatedSection(sections []Section, address uint64, flags SectionFlags) bool {
	for _, section := range sections {
		if section.Flags&SectionAlloc != 0 && section.Flags&flags == flags && rangeContains(section.Address, section.Size, address, 1) {
			return true
		}
	}
	return false
}

func stringInList(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func regionKindName(kind RegionKind) string {
	if kind == RegionFlash {
		return "flash"
	}
	if kind == RegionRAM {
		return "RAM"
	}
	return "reserved memory"
}
