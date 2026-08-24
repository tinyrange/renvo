// Package sam2695 drives the SAM2695 MIDI synthesizer used by the M5Stack
// Unit Synth.
package sam2695

type synthError string

func (e synthError) Error() string { return string(e) }

const (
	// ErrInvalidChannel reports a MIDI channel outside 0 through 15.
	ErrInvalidChannel synthError = "MIDI channel must be between 0 and 15"
	// ErrInvalidValue reports a MIDI data byte outside 0 through 127.
	ErrInvalidValue synthError = "MIDI data value must be between 0 and 127"
	// ErrShortWrite reports a transport that accepted only part of a message.
	ErrShortWrite synthError = "short MIDI write"
)

// Writer is the serial transport needed by Device.
type Writer interface {
	Write([]byte) (int, error)
}

// Device is one SAM2695 synthesizer connected over MIDI serial.
type Device struct {
	writer Writer
}

// New binds a synthesizer to a serial writer configured for 31,250 baud.
func New(writer Writer) *Device { return &Device{writer: writer} }

func (d *Device) write(message []byte) error {
	written, err := d.writer.Write(message)
	if err != nil {
		return err
	}
	if written != len(message) {
		return ErrShortWrite
	}
	return nil
}

func validChannel(channel byte) bool { return channel < 16 }
func validValue(value byte) bool     { return value < 128 }

// Reset sends the MIDI system reset message.
func (d *Device) Reset() error { return d.write([]byte{0xff}) }

// NoteOn starts pitch on channel with velocity. Middle C is pitch 60.
func (d *Device) NoteOn(channel, pitch, velocity byte) error {
	if !validChannel(channel) {
		return ErrInvalidChannel
	}
	if !validValue(pitch) || !validValue(velocity) {
		return ErrInvalidValue
	}
	return d.write([]byte{0x90 | channel, pitch, velocity})
}

// NoteOff releases pitch on channel.
func (d *Device) NoteOff(channel, pitch byte) error {
	if !validChannel(channel) {
		return ErrInvalidChannel
	}
	if !validValue(pitch) {
		return ErrInvalidValue
	}
	return d.write([]byte{0x80 | channel, pitch, 0})
}

// ControlChange sends a standard MIDI controller message.
func (d *Device) ControlChange(channel, controller, value byte) error {
	if !validChannel(channel) {
		return ErrInvalidChannel
	}
	if !validValue(controller) || !validValue(value) {
		return ErrInvalidValue
	}
	return d.write([]byte{0xb0 | channel, controller, value})
}

// SetInstrument selects bank and General MIDI program on channel.
func (d *Device) SetInstrument(bank, channel, program byte) error {
	if !validChannel(channel) {
		return ErrInvalidChannel
	}
	if !validValue(bank) || !validValue(program) {
		return ErrInvalidValue
	}
	if err := d.ControlChange(channel, 0, bank); err != nil {
		return err
	}
	return d.write([]byte{0xc0 | channel, program})
}

// SetChannelVolume changes the mix volume of channel.
func (d *Device) SetChannelVolume(channel, volume byte) error {
	return d.ControlChange(channel, 7, volume)
}

// SetPan places channel from hard left (0) through center (64) to hard right
// (127).
func (d *Device) SetPan(channel, pan byte) error {
	return d.ControlChange(channel, 10, pan)
}

// AllNotesOff releases every note currently active on channel.
func (d *Device) AllNotesOff(channel byte) error {
	return d.ControlChange(channel, 123, 0)
}

// SetMasterVolume changes the SAM2695 system volume using the universal MIDI
// master-volume SysEx message understood by the Unit Synth.
func (d *Device) SetMasterVolume(volume byte) error {
	if !validValue(volume) {
		return ErrInvalidValue
	}
	return d.write([]byte{0xf0, 0x7f, 0x7f, 0x04, 0x01, 0x00, volume, 0xf7})
}
