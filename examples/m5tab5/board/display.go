package board

const (
	clockBase = uintptr(0x500e6000)
	dsiBase   = uintptr(0x500a0000)
	pmuBase   = uintptr(0x50115000)
)

// DisplaySize is the native portrait scanout size. Applications rotate this
// to 1280 by 720 in landscape.
const (
	DisplayWidth  = 720
	DisplayHeight = 1280
)

func writePHY(address, value byte) {
	store32(dsiBase+0xb4, 0)
	store32(dsiBase+0xb8, 0x10000|uint32(address))
	store32(dsiBase+0xb4, 2)
	store32(dsiBase+0xb4, 0)
	store32(dsiBase+0xb8, uint32(value))
	store32(dsiBase+0xb4, 2)
	store32(dsiBase+0xb4, 0)
}

func enableDisplayPower() {
	// Tab5 connects the MIPI D-PHY rail to LDO channel 3. These are the
	// calibrated 2.5 V values reported by the ESP32-P4 v1.3 development unit.
	store32(pmuBase+0x1c0, 0x40200180)
	store32(pmuBase+0x1c4, 0xd5800000)
	delay(200000)
}

func enableDisplayClocks() {
	// DSI system clock, 20 MHz D-PHY config/reference clocks, and a 240 MHz
	// DPI source divided by three. The resulting 80 MHz clock is the integer
	// divider selected by ESP-IDF for the requested 70 MHz timing.
	update32(clockBase+0x18, 0, 1<<12)
	update32(clockBase+0x38, 3<<30, 0)
	update32(clockBase+0x3c, 0xffff, 0x2a3)
}

func initializePHY() bool {
	store32(dsiBase+0xa4, 1)
	store32(dsiBase+0x04, 1)
	store32(dsiBase+0xa0, 1)
	store32(dsiBase+0xa0, 3)
	store32(dsiBase+0xa0, 7)
	store32(dsiBase+0xa0, 0x0f)

	// 965 Mbps request on the legacy 20 MHz reference resolves to a 960 Mbps
	// lane clock (M=48, N=1). 0x1a is the 950-999 Mbps PHY range.
	writePHY(0x44, 0x34)
	writePHY(0x19, 0x30)
	writePHY(0x17, 0)
	writePHY(0x18, 0x0f)
	writePHY(0x18, 0x81)

	for count := 0; count < 5000000; count++ {
		if load32(dsiBase+0xb0)&0x95 == 0x95 {
			return true
		}
	}
	return false
}

func configureVideo() {
	// Host values correspond to the official Tab5 configuration: RGB565,
	// 720x1280, two lanes, and ST7121 portrait timing.
	store32(dsiBase+0x08, 0x00000c06)
	store32(dsiBase+0x0c, 0)
	store32(dsiBase+0x10, 0)
	store32(dsiBase+0x14, 0)
	store32(dsiBase+0x2c, 0x19)
	store32(dsiBase+0x34, 1)
	store32(dsiBase+0x38, 0x0000ff02)
	store32(dsiBase+0x3c, DisplayWidth)
	store32(dsiBase+0x40, 0)
	store32(dsiBase+0x44, 0)
	store32(dsiBase+0x48, 3)
	store32(dsiBase+0x4c, 60)
	store32(dsiBase+0x50, 1203)
	store32(dsiBase+0x54, 20)
	store32(dsiBase+0x58, 24)
	store32(dsiBase+0x5c, 200)
	store32(dsiBase+0x60, DisplayHeight)
	store32(dsiBase+0x68, 0x010f7f02)
	store32(dsiBase+0x94, 3)
	store32(dsiBase+0x98, 0x002e0080)
	store32(dsiBase+0x9c, 0x00320068)
	store32(dsiBase+0xa4, 0x00003f01)
	store32(dsiBase+0xf4, 6000)

}

func startPattern() {
	// Vertical color bars come from the DSI host itself. They deliberately
	// bypass the bridge and DW-GDMA so this probe tests only the panel link.
	update32(dsiBase+0x38, 0, 1<<16)
	store32(dsiBase+0x34, 0)
}

// InitPattern starts the DSI link and the controller's vertical color bars.
// It returns false if the D-PHY cannot lock or its three lanes do not stop.
func InitPattern() bool {
	if !InitPower() {
		print("TAB5 DISPLAY POWER FAIL\n")
		return false
	}
	if !resetPanelAndTouch() {
		print("TAB5 DISPLAY RESET FAIL\n")
		return false
	}
	enableDisplayPower()
	enableDisplayClocks()
	if !initializePHY() {
		print("TAB5 DISPLAY PHY FAIL\n")
		return false
	}
	configureVideo()
	initializeST7121()
	enableBacklight()
	startPattern()
	return true
}
