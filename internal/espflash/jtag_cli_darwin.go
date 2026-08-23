//go:build darwin

package espflash

func runJTAGCommand(args []string) (bool, int) {
	if len(args) < 2 || args[1] != "--probe-jtag" && args[1] != "--jtag" && args[1] != "--watch" &&
		args[1] != "--jtag-status" && args[1] != "--jtag-halt" && args[1] != "--jtag-resume" &&
		args[1] != "--jtag-regs" && args[1] != "--jtag-step" && args[1] != "--jtag-read" &&
		args[1] != "--jtag-set-pc" && args[1] != "--jtag-set-reg" && args[1] != "--jtag-blink" {
		return false, 0
	}
	print("renvoflash: direct ESP USB/JTAG is currently available on Linux only\n")
	return true, 1
}
