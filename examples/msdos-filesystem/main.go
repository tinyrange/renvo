package main

import "renvo.dev/device/dos"

func check(label string, err error) bool {
	if err == nil {
		return true
	}
	dos.WriteConsole(label + ": " + err.Error() + "\r\n")
	return false
}

func main() {
	const first = "RENVO.TXT"
	const second = "RENVO2.TXT"
	dos.WriteConsole("MS-DOS filesystem demo\r\n\r\n")
	_ = dos.Remove(first)
	_ = dos.Remove(second)

	file, err := dos.CreateFile(first, 0)
	if !check("create", err) {
		return
	}
	if !check("write", file.WriteAll([]byte("Written by Renvo on MS-DOS.\r\n"))) {
		return
	}
	if !check("close", file.Close()) {
		return
	}

	file, err = dos.OpenFile(first, dos.OpenRead)
	if !check("open", err) {
		return
	}
	buffer := make([]byte, 80)
	count, err := file.Read(buffer)
	if !check("read", err) {
		return
	}
	_ = file.Close()
	dos.WriteConsole("Read back: " + string(buffer[:count]))

	if !check("rename", dos.Rename(first, second)) {
		return
	}
	attributes, err := dos.FileAttributes(second)
	if !check("attributes", err) {
		return
	}
	if attributes&dos.AttributeArchive != 0 {
		dos.WriteConsole("Archive attribute is set.\r\n")
	}

	dos.WriteConsole("TXT files in this directory:\r\n")
	finder := dos.Find("*.TXT", dos.AttributeDirectory|dos.AttributeHidden|dos.AttributeSystem)
	for {
		entry, ok, findErr := finder.Next()
		if !check("find", findErr) || !ok {
			break
		}
		dos.WriteConsole("  " + entry.Name + "\r\n")
	}

	if check("remove", dos.Remove(second)) {
		dos.WriteConsole("Temporary file removed.\r\n")
	}
}
