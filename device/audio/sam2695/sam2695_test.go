package sam2695

import (
	"errors"
	"testing"
)

type recordingWriter struct {
	messages [][]byte
	n        int
	err      error
}

func (w *recordingWriter) Write(message []byte) (int, error) {
	w.messages = append(w.messages, append([]byte(nil), message...))
	if w.n != 0 || w.err != nil {
		return w.n, w.err
	}
	return len(message), nil
}

func TestNotesInstrumentAndControlsUseMidiMessages(t *testing.T) {
	writer := &recordingWriter{}
	synth := New(writer)
	operations := []func() error{
		func() error { return synth.Reset() },
		func() error { return synth.SetInstrument(0, 2, 4) },
		func() error { return synth.SetChannelVolume(2, 100) },
		func() error { return synth.SetPan(2, 64) },
		func() error { return synth.NoteOn(2, 60, 96) },
		func() error { return synth.NoteOff(2, 60) },
		func() error { return synth.AllNotesOff(2) },
		func() error { return synth.SetMasterVolume(80) },
	}
	for _, operation := range operations {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
	}
	want := [][]byte{
		{0xff},
		{0xb2, 0x00, 0x00}, {0xc2, 0x04},
		{0xb2, 0x07, 100}, {0xb2, 0x0a, 64},
		{0x92, 60, 96}, {0x82, 60, 0}, {0xb2, 123, 0},
		{0xf0, 0x7f, 0x7f, 0x04, 0x01, 0x00, 80, 0xf7},
	}
	if len(writer.messages) != len(want) {
		t.Fatalf("messages = %v", writer.messages)
	}
	for index := range want {
		if string(writer.messages[index]) != string(want[index]) {
			t.Fatalf("message %d = %v, want %v", index, writer.messages[index], want[index])
		}
	}
}

func TestInvalidMidiValuesAndShortWritesAreReported(t *testing.T) {
	synth := New(&recordingWriter{})
	if err := synth.NoteOn(16, 60, 100); !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("channel error = %v", err)
	}
	if err := synth.NoteOn(0, 128, 100); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("pitch error = %v", err)
	}
	if err := synth.SetMasterVolume(128); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("volume error = %v", err)
	}
	short := New(&recordingWriter{n: 1})
	if err := short.NoteOn(0, 60, 100); !errors.Is(err, ErrShortWrite) {
		t.Fatalf("short write error = %v", err)
	}
	transportError := synthError("transport failed")
	failing := New(&recordingWriter{err: transportError})
	if err := failing.Reset(); !errors.Is(err, transportError) {
		t.Fatalf("transport error = %v", err)
	}
}
