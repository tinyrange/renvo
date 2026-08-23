//go:build linux && (amd64 || renvo)

package espflash

import "os"

func runJTAGCommand(args []string) (bool, int) {
	if len(args) < 2 {
		return false, 0
	}
	mode := args[1]
	if isJTAGDebugMode(mode) {
		return true, runJTAGDebugCommand(args)
	}
	if mode != "--probe-jtag" && mode != "--jtag" && mode != "--watch" {
		return false, 0
	}
	if mode == "--probe-jtag" {
		if len(args) < 2 || len(args) > 3 {
			print("usage: renvoflash --probe-jtag [USB-DEVICE]\n")
			return true, 2
		}
		path := ""
		if len(args) == 3 {
			path = args[2]
		}
		debug, err := OpenESP32C6JTAG(path)
		if err != nil {
			print("renvoflash: " + err.Error() + "\n")
			return true, 1
		}
		status, statusErr := debug.dmiRead(dmiDMStatus)
		debug.Close()
		if statusErr != nil {
			print("renvoflash: DMI probe (abits " + decimal(debug.abits) + ", idle " + decimal(debug.idleClocks) + "): " + statusErr.Error() + "\n")
			return true, 1
		}
		print("ESP32-C6 USB/JTAG debug transport ready (dmstatus 0x" + hex(status) + ")\n")
		return true, 0
	}
	if len(args) < 3 || len(args) > 4 {
		print("usage: renvoflash " + mode + " ELF [USB-DEVICE]\n")
		return true, 2
	}
	path := ""
	if len(args) == 4 {
		path = args[3]
	}
	debug, err := OpenESP32C6JTAG(path)
	if err != nil {
		print("renvoflash: " + err.Error() + "\n")
		return true, 1
	}
	defer debug.Close()
	session := NewHotReloadSession(debug)
	var previousSource []byte
	var pendingSource []byte
	for {
		source, readErr := os.ReadFile(args[2])
		if readErr != nil {
			print("renvoflash: read " + args[2] + " failed\n")
			return true, 1
		}
		if !sameBytes(previousSource, source) {
			// A compiler may replace an ELF through several writes. After the
			// initial load, require two identical polls before parsing a change.
			if mode == "--watch" && len(previousSource) > 0 && !sameBytes(pendingSource, source) {
				pendingSource = append(pendingSource[:0], source...)
				sleep(10)
				continue
			}
			report, updateErr := session.Update(source)
			if updateErr != nil {
				print("renvoflash: " + updateErr.Error() + "\n")
				return true, 1
			}
			print("JTAG update: " + decimal(report.BytesWritten) + " bytes in " + decimal(report.PatchCount) + " patches\n")
			previousSource = append(previousSource[:0], source...)
			pendingSource = pendingSource[:0]
		}
		if mode != "--watch" {
			return true, 0
		}
		sleep(10)
	}
}

func sameBytes(first []byte, second []byte) bool {
	if len(first) != len(second) {
		return false
	}
	for i := 0; i < len(first); i++ {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}
