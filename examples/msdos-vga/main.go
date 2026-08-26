package main

import "renvo.dev/device/dos"

func main() {
	screen := dos.OpenVGA13()
	defer screen.Close()
	for color := 0; color < 64; color++ {
		screen.Palette(byte(color), byte(color*4), byte(255-color*4), byte(color*2))
	}
	line := make([]byte, 320)
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			line[x] = byte((x/5 + y/4) & 63)
		}
		screen.Blit(uint16(y)*320, line)
	}
	dos.WaitVerticalRetrace()
	dos.ReadKey()
}
