package espflash

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func TestParseLoadImageAndPlanPatches(t *testing.T) {
	firstELF := testELF(0x40824100, []testELFSegment{
		{address: 0x40824000, data: []byte{1, 2, 3, 4, 5, 6, 7, 8}, memorySize: 8, flags: 5},
		{address: 0x40800000, data: []byte{9, 10, 11, 12}, memorySize: 12, flags: 6},
	})
	first, err := ParseLoadImage(firstELF)
	if err != nil {
		t.Fatal(err)
	}
	if first.Entry != 0x40824100 || len(first.Segments) != 2 || first.Segments[0].Address != 0x40800000 {
		t.Fatalf("parsed image = %#v", first)
	}
	initial := PlanPatches(nil, first)
	if len(initial) != 2 || initial[0].Address != 0x40800000 || len(initial[0].Data) != 4 || len(initial[1].Data) != 8 {
		t.Fatalf("initial patches = %#v", initial)
	}

	secondELF := testELF(0x40824100, []testELFSegment{
		{address: 0x40824000, data: []byte{1, 2, 99, 4, 5, 6, 7, 8}, memorySize: 8, flags: 5},
		{address: 0x40800000, data: []byte{9, 10, 11, 12}, memorySize: 12, flags: 6},
	})
	second, err := ParseLoadImage(secondELF)
	if err != nil {
		t.Fatal(err)
	}
	delta := PlanPatches(&first, second)
	if len(delta) != 1 || delta[0].Address != 0x40824000 || !reflect.DeepEqual(delta[0].Data, []byte{1, 2, 99, 4}) {
		t.Fatalf("delta patches = %#v", delta)
	}
}

func TestPlanPatchesDoesNotRewriteUnchangedPaddedWord(t *testing.T) {
	image, err := ParseLoadImage(testELF(0x40824100, []testELFSegment{{
		address: 0x40824000, data: []byte{1, 2, 3, 4, 5}, memorySize: 8, flags: 5,
	}}))
	if err != nil {
		t.Fatal(err)
	}
	if patches := PlanPatches(&image, image); len(patches) != 0 {
		t.Fatalf("unchanged padded image patches = %#v, want none", patches)
	}
}

func TestParseLoadImageRejectsFlashLinkedELF(t *testing.T) {
	_, err := ParseLoadImage(testELF(0x42000100, []testELFSegment{{
		address: 0x42000000, data: []byte{1, 2, 3, 4}, memorySize: 4, flags: 5,
	}}))
	if err == nil || !strings.Contains(err.Error(), "esp32c6-jtag/riscv32") {
		t.Fatalf("error = %v", err)
	}
}

func TestPrepareImageRejectsJTAGLinkedELF(t *testing.T) {
	_, _, err := prepareImage(testELF(0x40824100, []testELFSegment{{
		address: 0x40824000, data: []byte{1, 2, 3, 4}, memorySize: 4, flags: 5,
	}}))
	if err == nil || !strings.Contains(err.Error(), "use --jtag") {
		t.Fatalf("error = %v", err)
	}
}

func TestHotReloadSessionWritesDeltaAndSkipsUnchangedImage(t *testing.T) {
	debug := &recordingDebugger{}
	session := NewHotReloadSession(debug)
	first := testELF(0x40824100, []testELFSegment{{
		address: 0x40824000, data: []byte{1, 2, 3, 4, 5, 6, 7, 8}, memorySize: 8, flags: 5,
	}})
	report, err := session.Update(first)
	if err != nil {
		t.Fatal(err)
	}
	if report.BytesWritten != 8 || strings.Join(debug.calls, ",") != "halt,write,fence,pc,resume" {
		t.Fatalf("first report/calls = %#v / %#v", report, debug.calls)
	}
	debug.calls = nil
	report, err = session.Update(first)
	if err != nil || !report.Unchanged || len(debug.calls) != 0 {
		t.Fatalf("unchanged report/calls/error = %#v / %#v / %v", report, debug.calls, err)
	}
	changed := testELF(0x40824100, []testELFSegment{{
		address: 0x40824000, data: []byte{1, 2, 3, 4, 5, 6, 42, 8}, memorySize: 8, flags: 5,
	}})
	report, err = session.Update(changed)
	if err != nil || report.BytesWritten != 4 || debug.writes[len(debug.writes)-1].Address != 0x40824004 {
		t.Fatalf("delta report/writes/error = %#v / %#v / %v", report, debug.writes, err)
	}
}

type recordingDebugger struct {
	calls  []string
	writes []Patch
}

func (debug *recordingDebugger) Halt() error { debug.calls = append(debug.calls, "halt"); return nil }
func (debug *recordingDebugger) WriteMemory(address uint32, data []byte) error {
	debug.calls = append(debug.calls, "write")
	debug.writes = append(debug.writes, Patch{Address: address, Data: append([]byte{}, data...)})
	return nil
}
func (debug *recordingDebugger) FenceI() error {
	debug.calls = append(debug.calls, "fence")
	return nil
}
func (debug *recordingDebugger) SetPC(address uint32) error {
	debug.calls = append(debug.calls, "pc")
	return nil
}
func (debug *recordingDebugger) Resume() error {
	debug.calls = append(debug.calls, "resume")
	return nil
}

type testELFSegment struct {
	address    uint32
	data       []byte
	memorySize uint32
	flags      uint32
}

func testELF(entry uint32, segments []testELFSegment) []byte {
	const headerSize = 52
	const programSize = 32
	offset := headerSize + programSize*len(segments)
	result := make([]byte, offset)
	copy(result, []byte{0x7f, 'E', 'L', 'F', 1, 1, 1})
	binary.LittleEndian.PutUint16(result[16:], 2)
	binary.LittleEndian.PutUint16(result[18:], 243)
	binary.LittleEndian.PutUint32(result[20:], 1)
	binary.LittleEndian.PutUint32(result[24:], entry)
	binary.LittleEndian.PutUint32(result[28:], headerSize)
	binary.LittleEndian.PutUint16(result[40:], headerSize)
	binary.LittleEndian.PutUint16(result[42:], programSize)
	binary.LittleEndian.PutUint16(result[44:], uint16(len(segments)))
	for i := 0; i < len(segments); i++ {
		at := headerSize + i*programSize
		binary.LittleEndian.PutUint32(result[at:], 1)
		binary.LittleEndian.PutUint32(result[at+4:], uint32(offset))
		binary.LittleEndian.PutUint32(result[at+8:], segments[i].address)
		binary.LittleEndian.PutUint32(result[at+12:], segments[i].address)
		binary.LittleEndian.PutUint32(result[at+16:], uint32(len(segments[i].data)))
		binary.LittleEndian.PutUint32(result[at+20:], segments[i].memorySize)
		binary.LittleEndian.PutUint32(result[at+24:], segments[i].flags)
		binary.LittleEndian.PutUint32(result[at+28:], 4)
		result = append(result, segments[i].data...)
		offset += len(segments[i].data)
	}
	return result
}
