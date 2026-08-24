package main

import (
	"renvo.dev/device/audio/sam2695"
	board "renvo.dev/device/board/m5nanoc6"
	"renvo.dev/device/uart"
)

const (
	leadChannel       = byte(0)
	bassChannel       = byte(1)
	arpeggioChannel   = byte(2)
	counterChannel    = byte(3)
	chordRootChannel  = byte(4)
	chordThirdChannel = byte(5)
	chordFifthChannel = byte(6)
	padRootChannel    = byte(7)
	padFifthChannel   = byte(8)
	drumChannel       = byte(9)
	padThirdChannel   = byte(10)
	accentChannel     = byte(11)

	stepMilliseconds = uint32(134) // About 112 BPM in sixteenth notes.
)

var (
	randomState uint32
	activeNotes [16]byte
	noteActive  [16]bool
	melodyIndex = 4
)

// Four familiar progressions provide a stable harmonic frame. A new one is
// selected each cycle while all of the playing patterns are generated live.
func chordRoot(progression, bar int) byte {
	switch progression & 3 {
	case 0: // C - Am - F - G
		return [4]byte{0, 9, 5, 7}[bar]
	case 1: // C - G - Am - F
		return [4]byte{0, 7, 9, 5}[bar]
	case 2: // Am - F - C - G
		return [4]byte{9, 5, 0, 7}[bar]
	default: // C - F - Am - G
		return [4]byte{0, 5, 9, 7}[bar]
	}
}

func nextRandom(limit uint32) uint32 {
	randomState ^= randomState << 13
	randomState ^= randomState >> 17
	randomState ^= randomState << 5
	if limit == 0 {
		return 0
	}
	return randomState % limit
}

func chordThird(root byte) byte {
	if root == 9 { // A minor
		return 3
	}
	return 4
}

func chordBase(root byte) byte {
	if root > 7 {
		return 48 + root
	}
	return 60 + root
}

func rootScaleIndex(root byte) int {
	switch root {
	case 5:
		return 3
	case 7:
		return 4
	case 9:
		return 5
	default:
		return 0
	}
}

func stopNote(synth *sam2695.Device, channel byte) {
	if noteActive[channel] {
		_ = synth.NoteOff(channel, activeNotes[channel])
		noteActive[channel] = false
	}
}

func startNote(synth *sam2695.Device, channel, pitch, velocity byte) {
	stopNote(synth, channel)
	_ = synth.NoteOn(channel, pitch, velocity)
	activeNotes[channel] = pitch
	noteActive[channel] = true
}

func setHarmony(synth *sam2695.Device, root byte) {
	base := chordBase(root)
	third := chordThird(root)
	// Electric-piano chord, spread in stereo.
	startNote(synth, chordRootChannel, base, 55)
	startNote(synth, chordThirdChannel, base+third, 50)
	startNote(synth, chordFifthChannel, base+7, 55)
	// A quieter pad an octave down gives the progression some depth.
	startNote(synth, padRootChannel, base-12, 34)
	startNote(synth, padThirdChannel, base+third-12, 30)
	startNote(synth, padFifthChannel, base+7-12, 34)
}

func playDrums(synth *sam2695.Device, step int, fill bool) {
	if step%2 == 0 {
		_ = synth.NoteOn(drumChannel, 42, 46) // Closed hi-hat.
	}
	if step == 0 || step == 8 || step == 10 {
		_ = synth.NoteOn(drumChannel, 36, 92) // Bass drum.
	}
	if step == 4 || step == 12 {
		_ = synth.NoteOn(drumChannel, 38, 88) // Snare.
	}
	if step == 14 {
		_ = synth.NoteOn(drumChannel, 46, 54) // Open hi-hat.
	}
	if fill && step >= 13 {
		_ = synth.NoteOn(drumChannel, byte(45+(step-13)*2), 70)
	}
}

func playArpeggio(synth *sam2695.Device, root byte, step int) {
	if step%2 != 0 || nextRandom(5) == 0 {
		stopNote(synth, arpeggioChannel)
		return
	}
	base := chordBase(root) + 12
	pattern := step/2 + int(nextRandom(2))
	switch pattern & 3 {
	case 0:
		startNote(synth, arpeggioChannel, base, 63)
	case 1:
		startNote(synth, arpeggioChannel, base+chordThird(root), 58)
	case 2:
		startNote(synth, arpeggioChannel, base+7, 62)
	default:
		startNote(synth, arpeggioChannel, base+12, 55)
	}
}

func playMelody(synth *sam2695.Device, root byte, step int) {
	if step%4 != 0 && !(step%2 == 0 && nextRandom(5) == 0) {
		return
	}
	if step == 0 {
		// Start each bar on a chord tone before resuming the random walk.
		melodyIndex = rootScaleIndex(root)
	} else {
		melodyIndex += int(nextRandom(3)) - 1
		if melodyIndex < 0 {
			melodyIndex = 0
		}
		if melodyIndex > 7 {
			melodyIndex = 7
		}
	}
	scale := [8]byte{72, 74, 76, 77, 79, 81, 83, 84}
	startNote(synth, leadChannel, scale[melodyIndex], byte(65+nextRandom(25)))
}

func playAccents(synth *sam2695.Device, root byte, step int) {
	if step == 6 || step == 14 {
		base := chordBase(root)
		startNote(synth, counterChannel, base+19, 45)
	} else if step == 0 || step == 8 {
		stopNote(synth, counterChannel)
	}
	if step == 3 || step == 11 {
		base := chordBase(root)
		startNote(synth, accentChannel, base+12+chordThird(root), 48)
	} else if step == 5 || step == 13 {
		stopNote(synth, accentChannel)
	}
}

func playCycle(synth *sam2695.Device) {
	progression := int(nextRandom(4))
	fillBar := int(nextRandom(4))
	for bar := 0; bar < 4; bar++ {
		root := chordRoot(progression, bar)
		setHarmony(synth, root)
		for step := 0; step < 16; step++ {
			if step == 0 || step == 8 || step == 12 && nextRandom(2) == 0 {
				bassPitch := byte(36 + root)
				if step == 12 {
					bassPitch += 7
				}
				startNote(synth, bassChannel, bassPitch, 82)
			}
			playDrums(synth, step, bar == fillBar)
			playArpeggio(synth, root, step)
			playMelody(synth, root, step)
			playAccents(synth, root, step)
			board.Clock.DelayMilliseconds(stepMilliseconds)
		}
	}
}

func configure(synth *sam2695.Device) {
	// General MIDI programs: flute lead, fingered bass, marimba, music box,
	// electric piano, warm pad, and nylon guitar.
	_ = synth.SetInstrument(0, leadChannel, 73)
	_ = synth.SetInstrument(0, bassChannel, 33)
	_ = synth.SetInstrument(0, arpeggioChannel, 12)
	_ = synth.SetInstrument(0, counterChannel, 10)
	_ = synth.SetInstrument(0, chordRootChannel, 4)
	_ = synth.SetInstrument(0, chordThirdChannel, 4)
	_ = synth.SetInstrument(0, chordFifthChannel, 4)
	_ = synth.SetInstrument(0, padRootChannel, 89)
	_ = synth.SetInstrument(0, padFifthChannel, 89)
	_ = synth.SetInstrument(0, padThirdChannel, 89)
	_ = synth.SetInstrument(0, accentChannel, 24)

	for channel := byte(0); channel < 12; channel++ {
		if channel != drumChannel {
			_ = synth.SetChannelVolume(channel, 92)
		}
	}
	_ = synth.SetChannelVolume(drumChannel, 82)
	_ = synth.SetPan(chordRootChannel, 28)
	_ = synth.SetPan(chordThirdChannel, 64)
	_ = synth.SetPan(chordFifthChannel, 100)
	_ = synth.SetPan(padRootChannel, 42)
	_ = synth.SetPan(padThirdChannel, 64)
	_ = synth.SetPan(padFifthChannel, 86)
	_ = synth.SetPan(arpeggioChannel, 38)
	_ = synth.SetPan(counterChannel, 94)
	_ = synth.SetPan(accentChannel, 82)
}

func main() {
	board.BlueLED.Set(false)
	board.Clock.DelayMilliseconds(500)

	synth := sam2695.New(uart.New(board.GroveUART, 31250))
	if err := synth.Reset(); err != nil {
		print("Unit Synth UART setup failed: ", err.Error(), "\n")
		for {
			board.Clock.DelayMilliseconds(1000)
		}
	}
	board.Clock.DelayMilliseconds(100)
	_ = synth.SetMasterVolume(88)
	configure(synth)
	randomState = board.Random.Uint32() ^ board.Clock.Ticks()
	if randomState == 0 {
		randomState = 1
	}
	print("Unit Synth ready; generating a twelve-channel groove\n")

	board.BlueLED.Set(true)
	for {
		playCycle(synth)
	}
}
