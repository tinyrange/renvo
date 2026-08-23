package espflash

// LoadSegment is one loadable region in a Renvo ESP32-C6 ELF image.
type LoadSegment struct {
	Address    uint32
	Data       []byte
	MemorySize uint32
	Executable bool
	Writable   bool
}

// LoadImage is the RAM-linked representation consumed by a hot-reload session.
type LoadImage struct {
	Entry    uint32
	Segments []LoadSegment
}

// Patch is one word-aligned memory update.
type Patch struct {
	Address uint32
	Data    []byte
}

// Debugger is the narrow target boundary required by the frontend. A concrete
// ESP32-C6 USB/JTAG debugger is provided on Linux; tests and other hosts can
// supply their own transport.
type Debugger interface {
	Halt() error
	WriteMemory(address uint32, data []byte) error
	FenceI() error
	SetPC(address uint32) error
	Resume() error
}

// ReloadReport describes one completed update.
type ReloadReport struct {
	Entry        uint32
	PatchCount   int
	BytesWritten int
	Unchanged    bool
}

// HotReloadSession retains the last linked image so subsequent updates write
// only changed words. Sessions are deliberately long-lived: keeping USB/JTAG
// claimed avoids process and probe setup costs in the edit/test loop.
type HotReloadSession struct {
	debug       Debugger
	previous    LoadImage
	hasPrevious bool
}

func NewHotReloadSession(debug Debugger) *HotReloadSession {
	return &HotReloadSession{debug: debug}
}

// Reset forgets the host-side image. The next Update writes every loadable
// byte, which is required after a board reset or power cycle.
func (session *HotReloadSession) Reset() {
	if session != nil {
		session.previous = LoadImage{}
		session.hasPrevious = false
	}
}

// Update parses a RAM-linked ELF, halts the core, applies its changed words,
// executes fence.i, redirects the PC to the image entry, and resumes it.
func (session *HotReloadSession) Update(elf []byte) (ReloadReport, error) {
	var report ReloadReport
	if session == nil || session.debug == nil {
		return report, fail("hot-reload debugger is unavailable")
	}
	next, err := ParseLoadImage(elf)
	if err != nil {
		return report, err
	}
	report.Entry = next.Entry
	var previous *LoadImage
	if session.hasPrevious {
		previous = &session.previous
	}
	patches := PlanPatches(previous, next)
	if len(patches) == 0 && session.hasPrevious && session.previous.Entry == next.Entry {
		report.Unchanged = true
		return report, nil
	}
	if err = session.debug.Halt(); err != nil {
		return report, err
	}
	for i := 0; i < len(patches); i++ {
		if err = session.debug.WriteMemory(patches[i].Address, patches[i].Data); err != nil {
			return report, err
		}
		report.PatchCount++
		report.BytesWritten += len(patches[i].Data)
	}
	if err = session.debug.FenceI(); err != nil {
		return report, err
	}
	if err = session.debug.SetPC(next.Entry); err != nil {
		return report, err
	}
	if err = session.debug.Resume(); err != nil {
		return report, err
	}
	session.previous = cloneLoadImage(next)
	session.hasPrevious = true
	return report, nil
}

// ParseLoadImage parses the ELF program headers used for JTAG loading. The
// hot-reload target is intentionally restricted to the ESP32-C6 HP SRAM window;
// ordinary XIP ELFs must be flashed instead of being written as memory.
func ParseLoadImage(source []byte) (LoadImage, error) {
	var image LoadImage
	if len(source) < 52 || source[0] != 0x7f || source[1] != 'E' || source[2] != 'L' || source[3] != 'F' || source[4] != 1 || source[5] != 1 {
		return image, fail("JTAG hot reload requires a little-endian ELF32 file")
	}
	if u16(source, 18) != 243 {
		return image, fail("JTAG hot reload requires a RISC-V ELF")
	}
	image.Entry = u32(source, 24)
	if !c6RAMAddress(image.Entry, 1) {
		return image, fail("ELF entry is not in ESP32-C6 SRAM; compile with target esp32c6-jtag/riscv32")
	}
	programAt := int(u32(source, 28))
	programSize := int(u16(source, 42))
	programCount := int(u16(source, 44))
	if programSize < 32 || programAt < 0 || programCount < 1 || programAt+programSize*programCount > len(source) {
		return image, fail("ELF program table is invalid")
	}
	for index := 0; index < programCount; index++ {
		at := programAt + index*programSize
		if u32(source, at) != 1 {
			continue
		}
		offset := int(u32(source, at+4))
		address := u32(source, at+8)
		fileSize := int(u32(source, at+16))
		memorySize := u32(source, at+20)
		flags := u32(source, at+24)
		if memorySize < uint32(fileSize) || offset < 0 || fileSize < 0 || offset+fileSize > len(source) {
			return LoadImage{}, fail("ELF load segment is invalid")
		}
		if memorySize == 0 {
			continue
		}
		if !c6RAMAddress(address, memorySize) {
			return LoadImage{}, fail("ELF contains a load segment outside ESP32-C6 SRAM")
		}
		segment := LoadSegment{
			Address: address, MemorySize: memorySize,
			Executable: flags&1 != 0, Writable: flags&2 != 0,
		}
		segment.Data = append(segment.Data, source[offset:offset+fileSize]...)
		image.Segments = append(image.Segments, segment)
	}
	if len(image.Segments) == 0 {
		return LoadImage{}, fail("ELF has no loadable segments")
	}
	sortLoadSegments(image.Segments)
	for i := 1; i < len(image.Segments); i++ {
		previousEnd := image.Segments[i-1].Address + image.Segments[i-1].MemorySize
		if image.Segments[i].Address < previousEnd {
			return LoadImage{}, fail("ELF load segments overlap")
		}
	}
	return image, nil
}

func c6RAMAddress(address uint32, size uint32) bool {
	const low = uint32(0x40800000)
	const high = uint32(0x40880000)
	return size > 0 && address >= low && address < high && size <= high-address
}

func sortLoadSegments(segments []LoadSegment) {
	for i := 1; i < len(segments); i++ {
		for j := i; j > 0 && segments[j].Address < segments[j-1].Address; j-- {
			segments[j], segments[j-1] = segments[j-1], segments[j]
		}
	}
}

// PlanPatches returns word-aligned changed ranges. Small unchanged runs are
// included between changes because one larger JTAG transaction is faster than
// several tiny transactions.
func PlanPatches(previous *LoadImage, next LoadImage) []Patch {
	var patches []Patch
	for segmentIndex := 0; segmentIndex < len(next.Segments); segmentIndex++ {
		segment := next.Segments[segmentIndex]
		dataSize := (len(segment.Data) + 3) &^ 3
		if previous == nil {
			data := make([]byte, dataSize)
			copy(data, segment.Data)
			if len(data) > 0 {
				patches = append(patches, Patch{Address: segment.Address, Data: data})
			}
			continue
		}
		start := -1
		lastChanged := -1
		for offset := 0; offset < dataSize; offset += 4 {
			changed := false
			for byteIndex := 0; byteIndex < 4; byteIndex++ {
				at := offset + byteIndex
				nextByte := byte(0)
				if at < len(segment.Data) {
					nextByte = segment.Data[at]
				}
				oldByte, found := loadImageByte(previous, segment.Address+uint32(at))
				if !found || oldByte != nextByte {
					changed = true
				}
			}
			if changed {
				if start < 0 {
					start = offset
				} else if lastChanged >= 0 && offset-lastChanged > 20 {
					patches = appendSegmentPatch(patches, segment, start, lastChanged+4)
					start = offset
				}
				lastChanged = offset
			}
		}
		if start >= 0 {
			patches = appendSegmentPatch(patches, segment, start, lastChanged+4)
		}
	}
	return patches
}

func appendSegmentPatch(patches []Patch, segment LoadSegment, start int, end int) []Patch {
	data := make([]byte, end-start)
	if start < len(segment.Data) {
		copyEnd := end
		if copyEnd > len(segment.Data) {
			copyEnd = len(segment.Data)
		}
		copy(data, segment.Data[start:copyEnd])
	}
	return append(patches, Patch{Address: segment.Address + uint32(start), Data: data})
}

func loadImageByte(image *LoadImage, address uint32) (byte, bool) {
	if image == nil {
		return 0, false
	}
	for i := 0; i < len(image.Segments); i++ {
		segment := image.Segments[i]
		paddedSize := uint32((len(segment.Data) + 3) &^ 3)
		if address < segment.Address || address >= segment.Address+paddedSize {
			continue
		}
		if address >= segment.Address+uint32(len(segment.Data)) {
			return 0, true
		}
		return segment.Data[address-segment.Address], true
	}
	return 0, false
}

func cloneLoadImage(image LoadImage) LoadImage {
	result := LoadImage{Entry: image.Entry, Segments: make([]LoadSegment, len(image.Segments))}
	for i := 0; i < len(image.Segments); i++ {
		result.Segments[i] = image.Segments[i]
		result.Segments[i].Data = append([]byte{}, image.Segments[i].Data...)
	}
	return result
}
