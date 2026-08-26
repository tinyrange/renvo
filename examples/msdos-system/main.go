package main

import "renvo.dev/device/dos"

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [6]byte
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[at:])
}

func two(value int) string {
	if value < 10 {
		return "0" + decimal(value)
	}
	return decimal(value)
}

func main() {
	major, minor := dos.Version()
	now := dos.Now()
	drive := dos.Drive()
	directory, err := dos.CurrentDirectory(0)

	dos.WriteConsole("MS-DOS system demo\r\n\r\n")
	dos.WriteConsole("DOS version: " + decimal(int(major)) + "." + two(int(minor)) + "\r\n")
	dos.WriteConsole("Date: " + decimal(now.Year) + "-" + two(now.Month) + "-" + two(now.Day) + "\r\n")
	dos.WriteConsole("Time: " + two(now.Hour) + ":" + two(now.Minute) + ":" + two(now.Second) + "\r\n")
	dos.WriteConsole("Drive: " + string([]byte{'A' + drive}) + ":\r\n")
	if err == nil {
		dos.WriteConsole("Directory: \\" + directory + "\r\n")
	}

	ticks, rollovers := dos.BIOSTicks()
	if ticks != 0 || rollovers != 0 {
		dos.WriteConsole("The BIOS clock is running.\r\n")
	}

	segment, allocErr := dos.AllocateSegment(4)
	if allocErr != nil {
		dos.WriteConsole("Conventional memory allocation failed: " + allocErr.Error() + "\r\n")
	} else {
		segment.Write(0, []byte("RENVO"))
		if segment.Load8(0) == 'R' && segment.Load8(4) == 'O' {
			dos.WriteConsole("Allocated and verified 64 bytes of conventional memory.\r\n")
		}
		_ = segment.Free()
	}

	line, modem := (dos.Serial{Port: 0}).Status()
	printer := (dos.Printer{Port: 0}).Status()
	if line != 0 || modem != 0 || printer != 0 {
		dos.WriteConsole("BIOS serial/printer status is available.\r\n")
	} else {
		dos.WriteConsole("No active serial or printer device reported.\r\n")
	}
}
