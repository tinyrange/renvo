package ssd1677

import (
	"errors"
	"testing"
)

type recordedTransaction struct {
	command byte
	length  int
	first   [8]byte
	last    byte
}

type fakeTransport struct {
	transactions []recordedTransaction
	current      recordedTransaction
	open         bool
	now          uint32
	busyUntil    uint32
	alwaysBusy   bool
	resets       int
	failData     bool
	endCalls     int
}

type fakeRefreshPoller struct {
	calls int
	err   error
}

func (poller *fakeRefreshPoller) PollDuringRefresh() error {
	poller.calls++
	return poller.err
}

func (transport *fakeTransport) Reset() error {
	transport.resets++
	return nil
}

func (transport *fakeTransport) Begin(command byte) error {
	if transport.open {
		return errors.New("nested transaction")
	}
	transport.open = true
	transport.current = recordedTransaction{command: command}
	return nil
}

func (transport *fakeTransport) Data(data []byte) error {
	if !transport.open {
		return errors.New("data outside transaction")
	}
	if transport.failData {
		transport.failData = false
		return errors.New("injected data failure")
	}
	for _, value := range data {
		if transport.current.length < len(transport.current.first) {
			transport.current.first[transport.current.length] = value
		}
		transport.current.last = value
		transport.current.length++
	}
	return nil
}

func (transport *fakeTransport) End() error {
	transport.endCalls++
	if !transport.open {
		return errors.New("end outside transaction")
	}
	transport.open = false
	transport.transactions = append(transport.transactions, transport.current)
	if transport.current.command == commandMasterActivation {
		transport.busyUntil = transport.now + 2
	}
	return nil
}

func (transport *fakeTransport) Busy() bool {
	return transport.alwaysBusy || transport.now < transport.busyUntil
}

func (transport *fakeTransport) Milliseconds() uint32 { return transport.now }

func (transport *fakeTransport) DelayMilliseconds(milliseconds uint32) {
	transport.now += milliseconds
}

func TestMonochromePacking(t *testing.T) {
	var frame Monochrome
	frame.Fill(true)
	if !frame.Set(0, 0, false) || !frame.Set(799, 479, false) {
		t.Fatal("Set rejected valid pixel")
	}
	if frame.Set(800, 0, false) || frame.Set(0, 480, false) {
		t.Fatal("Set accepted out-of-range pixel")
	}
	if frame[0] != 0x7f || frame[FrameSize-1] != 0xfe || frame[1] != 0xff {
		t.Fatalf("packed pixels = %02x/%02x/%02x", frame[0], frame[1], frame[FrameSize-1])
	}
}

func TestPackGrayRowUsesOTPPlaneEncoding(t *testing.T) {
	pixels := make([]Gray, Width)
	pixels[0] = White
	pixels[1] = LightGray
	pixels[2] = DarkGray
	pixels[3] = Black
	plane1 := make([]byte, BytesPerRow)
	plane2 := make([]byte, BytesPerRow)
	if err := PackGrayRow(pixels, plane1, plane2); err != nil {
		t.Fatal(err)
	}
	if plane1[0] != 0x50 || plane2[0] != 0x30 {
		t.Fatalf("gray planes = %02x/%02x, want 50/30", plane1[0], plane2[0])
	}
	pixels[4] = 4
	if err := PackGrayRow(pixels, plane1, plane2); err != ErrGrayLevel {
		t.Fatalf("invalid gray error = %v", err)
	}
}

func TestFullMonochromeInitializationWindowsAndRefresh(t *testing.T) {
	transport := &fakeTransport{}
	device := New(transport)
	frame := make([]byte, FrameSize)
	frame[0] = 0x12
	frame[FrameSize-1] = 0x34
	if err := device.FullMonochrome(frame); err != nil {
		t.Fatal(err)
	}
	if !device.HasBaseline() || device.PartialRefreshes() != 0 || transport.resets != 1 {
		t.Fatalf("state baseline=%v partial=%d resets=%d", device.HasBaseline(), device.PartialRefreshes(), transport.resets)
	}
	assertTransaction(t, transport.transactions, commandRAMXRange, []byte{0x00, 0x00, 0x1f, 0x03})
	assertTransaction(t, transport.transactions, commandDataEntryMode, []byte{0x01})
	assertTransaction(t, transport.transactions, commandRAMYRange, []byte{0xdf, 0x01, 0x00, 0x00})
	assertTransaction(t, transport.transactions, commandRAMYCounter, []byte{0xdf, 0x01})
	updates := transactionsFor(transport.transactions, commandUpdateControl)
	if len(updates) != 2 || updates[0].first[0] != 0xf8 || updates[1].first[0] != 0x14 {
		t.Fatalf("update controls = %+v", updates)
	}
	ram1 := transactionsFor(transport.transactions, commandWriteRAM1)
	ram2 := transactionsFor(transport.transactions, commandWriteRAM2)
	if len(ram1) != 2 || len(ram2) != 1 {
		t.Fatalf("RAM writes = RAM1:%d RAM2:%d", len(ram1), len(ram2))
	}
	if ram1[0].length != FrameSize || ram1[0].first[0] != ^byte(0x12) || ram1[0].last != ^byte(0x34) {
		t.Fatalf("inverted RAM1 = %+v", ram1[0])
	}
	if ram1[1].length != FrameSize || ram1[1].first[0] != 0x12 || ram2[0].length != FrameSize {
		t.Fatalf("baseline RAM writes = RAM1:%+v RAM2:%+v", ram1[1], ram2[0])
	}
	if transport.transactions[len(transport.transactions)-1].command != commandDeepSleep {
		t.Fatal("full refresh did not finish in deep sleep")
	}
}

func TestPartialRequiresBaselineAndRecoversAfterTenUpdates(t *testing.T) {
	transport := &fakeTransport{}
	device := New(transport)
	frame := make([]byte, FrameSize)
	if err := device.PartialMonochrome(frame); err != ErrNoBaseline {
		t.Fatalf("partial without baseline error = %v", err)
	}
	if err := device.FullMonochrome(frame); err != nil {
		t.Fatal(err)
	}
	for update := 0; update < 10; update++ {
		frame[0] = byte(update)
		if err := device.PartialMonochrome(frame); err != nil {
			t.Fatalf("partial %d: %v", update+1, err)
		}
	}
	if device.PartialRefreshes() != 10 {
		t.Fatalf("partial count = %d, want 10", device.PartialRefreshes())
	}
	before := len(transactionsFor(transport.transactions, commandSoftReset))
	if err := device.PartialMonochrome(frame); err != nil {
		t.Fatal(err)
	}
	after := len(transactionsFor(transport.transactions, commandSoftReset))
	if after != before+1 || device.PartialRefreshes() != 0 || !device.HasBaseline() {
		t.Fatalf("recovery resets=%d->%d partial=%d baseline=%v", before, after, device.PartialRefreshes(), device.HasBaseline())
	}
}

func TestFastMonochromeWritesNextAndSynchronizesPreviousPlane(t *testing.T) {
	transport := &fakeTransport{}
	device := New(transport)
	previous := make([]byte, FrameSize)
	previous[0] = 0x12
	previous[FrameSize-1] = 0x56
	if err := device.FullMonochrome(previous); err != nil {
		t.Fatal(err)
	}
	transport.transactions = nil
	next := make([]byte, FrameSize)
	next[0] = 0x34
	next[FrameSize-1] = 0x78
	if err := device.FastMonochrome(next, nil); err != nil {
		t.Fatal(err)
	}
	if !device.fastActive || device.PartialRefreshes() != 1 || !device.HasBaseline() {
		t.Fatalf("fast=%v partial=%d baseline=%v", device.fastActive, device.PartialRefreshes(), device.HasBaseline())
	}
	ram1 := transactionsFor(transport.transactions, commandWriteRAM1)
	ram2 := transactionsFor(transport.transactions, commandWriteRAM2)
	if len(ram1) != 1 || ram1[0].length != FrameSize || ram1[0].first[0] != 0x34 || ram1[0].last != 0x78 {
		t.Fatalf("next RAM plane = %+v", ram1)
	}
	if len(ram2) != 1 || ram2[0].length != FrameSize || ram2[0].first[0] != 0x34 || ram2[0].last != 0x78 {
		t.Fatalf("synchronized previous RAM plane = %+v", ram2)
	}
	lut := transactionsFor(transport.transactions, commandWriteLUT)
	if len(lut) != 1 || lut[0].length != 105 || lut[0].first[0] != fastDifferentialLUT[0] || lut[0].last != fastDifferentialLUT[104] {
		t.Fatalf("fast LUT write = %+v", lut)
	}
	assertTransaction(t, transport.transactions, commandGateVoltage, fastDifferentialLUT[105:106])
	assertTransaction(t, transport.transactions, commandSourceVoltage, fastDifferentialLUT[106:109])
	assertTransaction(t, transport.transactions, commandWriteVCOM, fastDifferentialLUT[109:])
	assertTransaction(t, transport.transactions, commandUpdateControl, []byte{0xcc})
	if transport.transactions[len(transport.transactions)-1].command == commandDeepSleep {
		t.Fatal("fast refresh entered deep sleep")
	}

	resets := transport.resets
	transport.transactions = nil
	if err := device.FastMonochrome(previous, nil); err != nil {
		t.Fatal(err)
	}
	if transport.resets != resets {
		t.Fatalf("active fast refresh reset controller %d -> %d", resets, transport.resets)
	}
	assertTransaction(t, transport.transactions, commandUpdateControl, []byte{0x0c})
}

func TestFastMonochromeRequiresBaseline(t *testing.T) {
	device := New(&fakeTransport{})
	frame := make([]byte, FrameSize)
	if err := device.FastMonochrome(frame, nil); err != ErrNoBaseline {
		t.Fatalf("FastMonochrome error = %v", err)
	}
}

func TestRefreshPollerRunsWhileWaveformIsBusy(t *testing.T) {
	transport := &fakeTransport{}
	device := New(transport)
	frame := make([]byte, FrameSize)
	poller := &fakeRefreshPoller{}
	if err := device.fullMonochrome(frame, poller); err != nil {
		t.Fatal(err)
	}
	if poller.calls < 2 {
		t.Fatalf("full refresh poll calls = %d", poller.calls)
	}
	poller.calls = 0
	if err := device.FastMonochrome(frame, poller); err != nil {
		t.Fatal(err)
	}
	if poller.calls == 0 {
		t.Fatal("fast refresh did not poll while busy")
	}
}

func TestFullGrayUsesReversedGateWindowAndInvalidatesBaseline(t *testing.T) {
	transport := &fakeTransport{}
	device := New(transport)
	plane1 := make([]byte, FrameSize)
	plane2 := make([]byte, FrameSize)
	if err := device.FullMonochrome(plane1); err != nil {
		t.Fatal(err)
	}
	transport.transactions = nil
	if err := device.FullGray(plane1, plane2); err != nil {
		t.Fatal(err)
	}
	if device.HasBaseline() {
		t.Fatal("gray refresh retained monochrome baseline")
	}
	assertTransaction(t, transport.transactions, commandDataEntryMode, []byte{0x01})
	assertTransaction(t, transport.transactions, commandRAMXRange, []byte{0x00, 0x00, 0x1f, 0x03})
	assertTransaction(t, transport.transactions, commandRAMYRange, []byte{0xdf, 0x01, 0x00, 0x00})
	assertTransaction(t, transport.transactions, commandUpdateControl, []byte{0xd7})
	if got := transactionsFor(transport.transactions, commandWriteRAM1); len(got) != 1 || got[0].length != FrameSize {
		t.Fatalf("gray RAM1 writes = %+v", got)
	}
	if got := transactionsFor(transport.transactions, commandWriteRAM2); len(got) != 1 || got[0].length != FrameSize {
		t.Fatalf("gray RAM2 writes = %+v", got)
	}
	if err := device.PartialMonochrome(plane1); err != ErrNoBaseline {
		t.Fatalf("partial after gray error = %v", err)
	}
}

type testGraySource struct{}

func (testGraySource) FillGrayRow(plane, row int, destination []byte) error {
	for index := range destination {
		destination[index] = byte(plane*2 + row&1)
	}
	return nil
}

func TestFullGrayStreamingWritesTwoCompletePackedPlanes(t *testing.T) {
	transport := &fakeTransport{}
	device := New(transport)
	if err := device.FullGrayStream(testGraySource{}); err != nil {
		t.Fatal(err)
	}
	ram1 := transactionsFor(transport.transactions, commandWriteRAM1)
	ram2 := transactionsFor(transport.transactions, commandWriteRAM2)
	if len(ram1) != 1 || ram1[0].length != FrameSize || ram1[0].first[0] != 0 || ram1[0].last != 1 {
		t.Fatalf("streamed RAM1 = %+v", ram1)
	}
	if len(ram2) != 1 || ram2[0].length != FrameSize || ram2[0].first[0] != 2 || ram2[0].last != 3 {
		t.Fatalf("streamed RAM2 = %+v", ram2)
	}
}

func TestBusyTimeoutIsBounded(t *testing.T) {
	transport := &fakeTransport{alwaysBusy: true}
	device := New(transport)
	if err := device.SetBusyTimeout(3); err != nil {
		t.Fatal(err)
	}
	if err := device.FullMonochrome(make([]byte, FrameSize)); err != ErrTimeout {
		t.Fatalf("busy error = %v, want %v", err, ErrTimeout)
	}
	if transport.now > 5 {
		t.Fatalf("timeout consumed %d ms", transport.now)
	}
}

func TestDataFailureEndsTransaction(t *testing.T) {
	transport := &fakeTransport{failData: true}
	device := New(transport)
	if err := device.FullMonochrome(make([]byte, FrameSize)); err == nil {
		t.Fatal("data failure was ignored")
	}
	if transport.open || transport.endCalls != len(transport.transactions) {
		t.Fatalf("failed transaction open=%v endCalls=%d transactions=%d", transport.open, transport.endCalls, len(transport.transactions))
	}
}

func assertTransaction(t *testing.T, transactions []recordedTransaction, command byte, data []byte) {
	t.Helper()
	matches := transactionsFor(transactions, command)
	for _, transaction := range matches {
		if transaction.length != len(data) {
			continue
		}
		equal := true
		for index, value := range data {
			if transaction.first[index] != value {
				equal = false
				break
			}
		}
		if equal {
			return
		}
	}
	t.Fatalf("command %02x data %x not found in %+v", command, data, matches)
}

func transactionsFor(transactions []recordedTransaction, command byte) []recordedTransaction {
	var matches []recordedTransaction
	for _, transaction := range transactions {
		if transaction.command == command {
			matches = append(matches, transaction)
		}
	}
	return matches
}
