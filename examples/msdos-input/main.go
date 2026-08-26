package main

import "renvo.dev/device/dos"

func delay(ticks uint32) {
	start, _ := dos.BIOSTicks()
	for {
		now, _ := dos.BIOSTicks()
		if now-start >= ticks {
			return
		}
	}
}

func tone(frequency int) {
	dos.SpeakerOn(frequency)
	delay(2)
	dos.SpeakerOff()
}

func main() {
	dos.WriteConsole("MS-DOS input and PC hardware demo\r\n")
	dos.WriteConsole("Playing the PC speaker, then waiting for a key.\r\n")
	tone(262)
	tone(330)
	tone(392)

	mouse, found := dos.OpenMouse()
	if found {
		mouse.SetBounds(0, 639, 0, 199)
		mouse.Show()
		x, y, buttons := mouse.Position()
		dos.SetCursor(0, x/8, y/8)
		if buttons != 0 {
			dos.WriteConsole("A mouse button is held.\r\n")
		}
		dos.SetCursor(0, 0, 4)
		dos.WriteConsole("Mouse driver detected.\r\n")
	} else {
		dos.WriteConsole("No mouse driver detected.\r\n")
	}

	dos.WriteConsole("Press any key to finish... ")
	key := dos.ReadKey()
	if found {
		mouse.Hide()
	}
	if key.ASCII >= 32 && key.ASCII < 127 {
		dos.Teletype(0, 7, key.ASCII)
	}
	dos.WriteConsole("\r\n")
}
