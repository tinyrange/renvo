package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/motor/rollercan"
)

// These are phase-current and speed targets, not USB power guarantees.
const currentLimit = int32(10000) // 100 mA phase current.
const speedTarget = int32(1000)   // 10 RPM.

func halt(motor *rollercan.Device, err error) {
	// A broken bus may prevent this stop command from reaching the motor.
	stopErr := motor.SetOutput(false)
	board.RGB.Set(32, 0, 0)
	print("RollerCAN: ", err.Error(), "\n")
	if stopErr != nil {
		print("Stop command failed: ", stopErr.Error(), "; disconnect motor power\n")
	}
	for {
		board.Clock.DelayMilliseconds(1000)
	}
}

func configure(motor *rollercan.Device) error {
	if err := motor.SetOutput(false); err != nil {
		return err
	}
	if err := motor.SetStallProtection(true); err != nil {
		return err
	}
	if err := motor.SetPositionProtection(true); err != nil {
		return err
	}
	if err := motor.SetButtonModeSwitch(false); err != nil {
		return err
	}
	if err := motor.SetMode(rollercan.SpeedMode); err != nil {
		return err
	}
	if err := motor.SetSpeedMaxCurrent(currentLimit); err != nil {
		return err
	}
	return motor.SetSpeedTarget(speedTarget)
}

func main() {
	motor := rollercan.New(i2c.New(board.Grove))
	if err := configure(motor); err != nil {
		halt(motor, err)
		return
	}
	version, err := motor.FirmwareVersion()
	if err != nil {
		halt(motor, err)
		return
	}
	print("RollerCAN firmware v", version, "; hold NanoC6 button for 10 RPM, release to stop\n")
	// Require a release after boot before accepting a held button.
	for board.Button.Pressed() {
		board.Clock.DelayMilliseconds(20)
	}
	wasPressed := false
	ticks := 0
	for {
		pressed := board.Button.Pressed()
		if pressed != wasPressed {
			if err := motor.SetOutput(pressed); err != nil {
				halt(motor, err)
				return
			}
			wasPressed = pressed
		}
		faults, err := motor.Faults()
		if err != nil {
			halt(motor, err)
			return
		}
		if faults != 0 {
			if err := motor.SetOutput(false); err != nil {
				halt(motor, err)
				return
			}
			print("RollerCAN fault mask: ", byte(faults), "; output disabled; restart after resolving fault\n")
			board.RGB.Set(32, 0, 0)
			for {
				board.Clock.DelayMilliseconds(1000)
			}
		}
		if pressed {
			board.RGB.Set(0, 24, 0)
		} else {
			board.RGB.Set(0, 0, 16)
		}
		if ticks == 0 {
			speed, err := motor.Speed()
			if err != nil {
				halt(motor, err)
				return
			}
			current, err := motor.Current()
			if err != nil {
				halt(motor, err)
				return
			}
			voltage, err := motor.Voltage()
			if err != nil {
				halt(motor, err)
				return
			}
			print("Speed_cRPM:", speed, "\tPhaseCurrent_cMA:", current, "\tVoltage_cV:", voltage, "\n")
		}
		ticks = (ticks + 1) % 25
		board.Clock.DelayMilliseconds(20)
	}
}
