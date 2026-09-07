package main

import "renvo.dev/device/motor/rollercan"

type bus struct {
	reg     [256]byte
	address uint16
	writes  int
	delay   uint32
	fail    bool
}

func (b *bus) Tx(address uint16, write, read []byte) error {
	if b.fail {
		return rollercan.ErrRange
	}
	b.address = address
	reg := int(write[0])
	if len(read) > 0 {
		copy(read, b.reg[reg:reg+len(read)])
	} else {
		b.writes++
		copy(b.reg[reg:reg+len(write)-1], write[1:])
	}
	return nil
}
func (b *bus) DelayMilliseconds(ms uint32) { b.delay += ms }

var checkNumber int

func check(ok bool) {
	checkNumber++
	if !ok {
		print("failed check ", checkNumber, "\n")
		panic("rollercan protocol regression")
	}
}

func main() {
	b := &bus{}
	b.reg[0xfe] = 3
	d := rollercan.New(b)
	check(b.writes == 0)
	check(d.SetSpeedTarget(-12345) == nil)
	check(b.reg[0x40] == 0xc7 && b.reg[0x43] == 0xff)
	v, err := d.SpeedTarget()
	check(err == nil && v == -12345)
	check(d.SetPositionTarget(-36000) == nil)
	v, err = d.PositionTarget()
	check(err == nil && v == -36000)
	check(d.SetSpeedMaxCurrent(10000) == nil)
	check(d.SetPositionMaxCurrent(10000) == nil)
	check(d.SetCurrentTarget(-10000) == nil)
	v, err = d.CurrentTarget()
	check(err == nil && v == -10000)
	pid := rollercan.PID{P: 100000, I: 10000000, D: 0xffffffff}
	check(d.SetSpeedPID(pid) == nil && d.SetPositionPID(pid) == nil)
	p, err := d.SpeedPID()
	check(err == nil && p.P == 100000 && p.I == 10000000 && p.D == 0xffffffff)
	p, err = d.PositionPID()
	check(err == nil && p.D == 0xffffffff)
	check(d.SetRGB(1, 2, 3) == nil && b.reg[0x30] == 3)
	r, g, blue, err := d.RGB()
	check(err == nil && r == 1 && g == 2 && blue == 3)
	check(d.SetRGBMode(rollercan.RGBUser) == nil)
	check(d.SetBrightness(10) == nil)
	check(d.SetMode(rollercan.SpeedMode) == nil && d.SetOutput(false) == nil)
	check(d.SetStallProtection(true) == nil && d.SetPositionProtection(true) == nil)
	check(d.SetButtonModeSwitch(false) == nil && d.ClearStall() == nil)
	check(d.SetCANID(255) == nil && d.SetCANBaudRate(rollercan.CAN125Kbps) == nil)
	check(d.SetDialCounter(-2147483648) == nil)
	v, err = d.DialCounter()
	check(err == nil && v == -2147483648)
	check(d.SetExtendedPositionTarget(rollercan.ExtendedPosition{Turns: -1, Angle: 35900}) == nil)
	position, err := d.ExtendedPositionTarget()
	check(err == nil && position.Turns == -1 && position.Angle == 35900)
	b.reg[0xe0], b.reg[0xeb] = 23, 42
	uid, err := d.UID()
	check(err == nil && uid[0] == 23 && uid[11] == 42)
	check(d.SaveConfig() == nil && b.delay == 100)
	check(d.SetI2CAddress(0x65) == nil && b.address == 0x64)
	_, err = d.FirmwareVersion()
	check(err == nil && b.address == 0x65 && d.Address() == 0x65)
	before := b.writes
	check(d.SetCurrentTarget(120001) == rollercan.ErrRange && b.writes == before)
	b.reg[0xfe] = 2
	_, err = d.UID()
	check(err == rollercan.ErrFirmware)
	b.fail = true
	check(d.SetI2CAddress(0x66) == rollercan.ErrRange && d.Address() == 0x65)
	v, err = d.SpeedTarget()
	check(err == rollercan.ErrRange && v == 0)
	print("PASS\n")
}
