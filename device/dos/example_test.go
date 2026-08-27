package dos_test

import (
	"fmt"

	"renvo.dev/device/dos"
)

// Now combines the DOS date and time services into one value.
func ExampleNow() {
	now := dos.Now()
	fmt.Printf("%04d-%02d-%02d %02d:%02d:%02d\n",
		now.Year, now.Month, now.Day,
		now.Hour, now.Minute, now.Second,
	)
}

func ExampleWriteConsole() {
	_ = dos.WriteConsole("Hello from Renvo!\r\n")
}

func ExampleFinder_Next() {
	finder := dos.Find("*.TXT", dos.AttributeArchive)
	for {
		entry, ok, err := finder.Next()
		if err != nil || !ok {
			break
		}
		_ = dos.WriteConsole(entry.Name + "\r\n")
	}
}

func ExampleFile_WriteAll() {
	file, err := dos.CreateFile("HELLO.TXT", dos.AttributeArchive)
	if err != nil {
		return
	}
	defer file.Close()

	_ = file.WriteAll([]byte("Hello from Renvo!\r\n"))
}

func ExampleVGA13_Set() {
	display := dos.OpenVGA13()
	defer display.Close()

	display.Clear(0)
	display.Palette(1, 255, 96, 32)
	display.Set(160, 100, 1)
}

func ExampleChangeDir() {
	if err := dos.ChangeDir("C:\\GAMES"); err != nil {
		_ = dos.WriteConsole("directory not found\r\n")
	}
}

func ExampleCurrentDirectory() {
	path, err := dos.CurrentDirectory(0) // Zero selects the current drive.
	if err == nil {
		_ = dos.WriteConsole(path + "\r\n")
	}
}

func ExampleFileAttributes() {
	attributes, err := dos.FileAttributes("README.TXT")
	if err == nil && attributes&dos.AttributeReadOnly != 0 {
		_ = dos.WriteConsole("read-only\r\n")
	}
}

func ExampleMakeDir() {
	if err := dos.MakeDir("OUTPUT"); err != nil {
		_ = dos.WriteConsole("could not create OUTPUT\r\n")
	}
}

func ExampleOpenFile() {
	file, err := dos.OpenFile("CONFIG.INI", dos.OpenRead)
	if err != nil {
		return
	}
	defer file.Close()

	buffer := make([]byte, 128)
	count, _ := file.Read(buffer)
	_ = buffer[:count]
}

func ExampleReadKey() {
	_ = dos.WriteConsole("Press a key: ")
	key := dos.ReadKey()
	if key.ASCII != 0 {
		_ = dos.WriteConsole(string([]byte{key.ASCII}) + "\r\n")
	}
}

func ExampleRename() {
	if err := dos.Rename("OLD.TXT", "NEW.TXT"); err != nil {
		_ = dos.WriteConsole("rename failed\r\n")
	}
}

func ExampleSetCursor() {
	dos.SetCursor(0, 10, 5)
	dos.Teletype(0, 7, '>')
}

func ExampleSetDrive() {
	available := dos.SetDrive(2) // Select drive C: (A: is zero).
	_ = available
}

func ExampleSetFileAttributes() {
	_ = dos.SetFileAttributes("CONFIG.INI", dos.AttributeReadOnly|dos.AttributeArchive)
}

func ExampleSpeakerOn() {
	dos.SpeakerOn(440)
	// Keep the tone active while doing work or waiting.
	dos.SpeakerOff()
}

func ExampleVersion() {
	major, minor := dos.Version()
	fmt.Printf("MS-DOS %d.%d\n", major, minor)
}

func ExampleMouse_Position() {
	mouse, ok := dos.OpenMouse()
	if !ok {
		return
	}
	mouse.Show()
	defer mouse.Hide()

	x, y, buttons := mouse.Position()
	_, _, _ = x, y, buttons
}

func ExampleMouse_SetBounds() {
	mouse, ok := dos.OpenMouse()
	if !ok {
		return
	}
	mouse.SetBounds(0, 319, 0, 199)
}

func ExamplePrinter_Write() {
	printer := dos.Printer{Port: 0}
	status := printer.Write('R')
	_ = status
}

func ExampleSerial_Configure() {
	serial := dos.Serial{Port: 0}
	status := serial.Configure(dos.Serial9600 | dos.Serial8Bits)
	_ = status
}

func ExampleSerial_Write() {
	serial := dos.Serial{Port: 0}
	status := serial.Write('R')
	_ = status
}

func ExampleSegment_Write() {
	segment, err := dos.AllocateSegment(256) // Allocate 4 KiB.
	if err != nil {
		return
	}
	defer segment.Free()

	segment.Write(0, []byte("sprite data"))
}

func ExampleVGA13_Blit() {
	display := dos.OpenVGA13()
	defer display.Close()

	scanline := make([]byte, 320)
	display.Blit(100*320, scanline)
}

func ExampleVGAPlanar_Set() {
	display := dos.OpenVGAPlanar()
	defer display.Close()

	for x := 40; x < 200; x++ {
		display.Set(x, 80, 12)
	}
}
