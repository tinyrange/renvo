//go:build m5papermonolite

package board

import (
	"errors"
	"testing"

	"renvo.dev/device/gpio"
	"renvo.dev/device/ioexpander/m5ioe1"
)

type fakePowerPMIC struct {
	identityCalls   int
	initializeCalls int
	err             error
}

func (p *fakePowerPMIC) Identify() (uint16, error) {
	p.identityCalls++
	return 0x2050, p.err
}

func (p *fakePowerPMIC) Initialize() error {
	p.initializeCalls++
	return p.err
}

type powerAction struct {
	operation byte
	pin       m5ioe1.Pin
	level     bool
}

type fakePowerIOExpander struct {
	identityCalls   int
	initializeCalls int
	actions         []powerAction
	levels          [15]bool
	failPin         m5ioe1.Pin
	failLevel       bool
	failOnce        bool
	mismatchPin     m5ioe1.Pin
}

func (e *fakePowerIOExpander) Identify() (uint16, error) {
	e.identityCalls++
	return 0x1234, nil
}

func (e *fakePowerIOExpander) Initialize() error {
	e.initializeCalls++
	return nil
}

func (e *fakePowerIOExpander) ConfigureOutput(pin m5ioe1.Pin, level bool) error {
	e.actions = append(e.actions, powerAction{operation: 'c', pin: pin, level: level})
	e.levels[pin] = level
	return nil
}

func (e *fakePowerIOExpander) SetOutput(pin m5ioe1.Pin, level bool) error {
	e.actions = append(e.actions, powerAction{operation: 's', pin: pin, level: level})
	if e.failOnce && pin == e.failPin && level == e.failLevel {
		e.failOnce = false
		return errors.New("injected output failure")
	}
	e.levels[pin] = level
	return nil
}

func (e *fakePowerIOExpander) Output(pin m5ioe1.Pin) (bool, error) {
	e.actions = append(e.actions, powerAction{operation: 'r', pin: pin})
	level := e.levels[pin]
	if pin == e.mismatchPin {
		level = !level
	}
	return level, nil
}

type fakePowerPin struct {
	setBeforeConfigure bool
	level              bool
	configured         int
}

func (p *fakePowerPin) Set(level bool) {
	if p.configured == 0 {
		p.setBeforeConfigure = true
	}
	p.level = level
}

func (p *fakePowerPin) Configure(config gpio.Config) error {
	if config.Direction != gpio.Output {
		return errors.New("chip select was not configured as output")
	}
	p.configured++
	return nil
}

type fakePowerDelay struct {
	delays []uint32
}

func (d *fakePowerDelay) DelayMilliseconds(milliseconds uint32) {
	d.delays = append(d.delays, milliseconds)
}

func TestPowerProbeIsReadOnly(t *testing.T) {
	pmic := &fakePowerPMIC{}
	expander := &fakePowerIOExpander{}
	chipSelect := &fakePowerPin{}
	delay := &fakePowerDelay{}
	power := newPowerDevice(pmic, expander, chipSelect, delay)
	identity, err := power.Probe()
	if err != nil || identity.PMIC != 0x2050 || identity.IOExpander != 0x1234 {
		t.Fatalf("Probe() = %+v, %v", identity, err)
	}
	if pmic.identityCalls != 1 || pmic.initializeCalls != 0 ||
		expander.identityCalls != 1 || expander.initializeCalls != 0 ||
		len(expander.actions) != 0 || chipSelect.configured != 0 || len(delay.delays) != 0 {
		t.Fatalf("probe mutated hardware: pmic=%+v expander=%+v cs=%+v delays=%v", pmic, expander, chipSelect, delay.delays)
	}
}

func TestPowerEnableAndDisableSequence(t *testing.T) {
	pmic := &fakePowerPMIC{}
	expander := &fakePowerIOExpander{}
	chipSelect := &fakePowerPin{}
	delay := &fakePowerDelay{}
	power := newPowerDevice(pmic, expander, chipSelect, delay)
	if err := power.EnableDisplayAndTouch(); err != nil {
		t.Fatal(err)
	}
	wantEnable := []powerAction{
		{operation: 'c', pin: m5ioe1.Pin5},
		{operation: 'c', pin: m5ioe1.Pin6},
		{operation: 'c', pin: m5ioe1.Pin3},
		{operation: 'c', pin: m5ioe1.Pin13},
		{operation: 's', pin: m5ioe1.Pin3, level: true},
		{operation: 's', pin: m5ioe1.Pin13, level: true},
		{operation: 's', pin: m5ioe1.Pin5, level: true},
		{operation: 's', pin: m5ioe1.Pin6, level: true},
		{operation: 'r', pin: m5ioe1.Pin3},
		{operation: 'r', pin: m5ioe1.Pin13},
		{operation: 'r', pin: m5ioe1.Pin5},
		{operation: 'r', pin: m5ioe1.Pin6},
	}
	assertPowerActions(t, expander.actions, wantEnable)
	if !chipSelect.setBeforeConfigure || !chipSelect.level || chipSelect.configured != 1 {
		t.Fatalf("chip select = %+v", chipSelect)
	}
	if len(delay.delays) != 2 || delay.delays[0] != 8 || delay.delays[1] != 2 {
		t.Fatalf("enable delays = %v", delay.delays)
	}

	expander.actions = nil
	if err := power.DisableDisplayAndTouch(); err != nil {
		t.Fatal(err)
	}
	wantDisable := []powerAction{
		{operation: 's', pin: m5ioe1.Pin5},
		{operation: 's', pin: m5ioe1.Pin6},
		{operation: 's', pin: m5ioe1.Pin13},
		{operation: 's', pin: m5ioe1.Pin3},
		{operation: 'r', pin: m5ioe1.Pin3},
		{operation: 'r', pin: m5ioe1.Pin13},
		{operation: 'r', pin: m5ioe1.Pin5},
		{operation: 'r', pin: m5ioe1.Pin6},
	}
	assertPowerActions(t, expander.actions, wantDisable)
	if len(delay.delays) != 3 || delay.delays[2] != 2 {
		t.Fatalf("all delays = %v", delay.delays)
	}
}

func TestPowerEnableFailureAttemptsCompleteShutdown(t *testing.T) {
	pmic := &fakePowerPMIC{}
	expander := &fakePowerIOExpander{failPin: m5ioe1.Pin13, failLevel: true, failOnce: true}
	power := newPowerDevice(pmic, expander, &fakePowerPin{}, &fakePowerDelay{})
	if err := power.EnableDisplayAndTouch(); err == nil {
		t.Fatal("EnableDisplayAndTouch() succeeded after injected failure")
	}
	for _, pin := range []m5ioe1.Pin{m5ioe1.Pin3, m5ioe1.Pin5, m5ioe1.Pin6, m5ioe1.Pin13} {
		if expander.levels[pin] {
			t.Fatalf("pin %d remained high after rollback", pin)
		}
	}
	if power.active {
		t.Fatal("power remained active after rollback")
	}
}

func TestPowerDisableBeforeEnableEstablishesPersistentSafeState(t *testing.T) {
	pmic := &fakePowerPMIC{}
	expander := &fakePowerIOExpander{}
	chipSelect := &fakePowerPin{}
	power := newPowerDevice(pmic, expander, chipSelect, &fakePowerDelay{})
	if err := power.DisableDisplayAndTouch(); err != nil {
		t.Fatal(err)
	}
	if pmic.initializeCalls != 1 || expander.initializeCalls != 1 {
		t.Fatalf("shutdown did not initialize devices: pmic=%d expander=%d", pmic.initializeCalls, expander.initializeCalls)
	}
	for _, pin := range []m5ioe1.Pin{m5ioe1.Pin3, m5ioe1.Pin5, m5ioe1.Pin6, m5ioe1.Pin13} {
		if expander.levels[pin] {
			t.Fatalf("pin %d remained high after initial shutdown", pin)
		}
	}
	if !chipSelect.level || chipSelect.configured != 1 {
		t.Fatalf("chip select = %+v", chipSelect)
	}
}

func TestPowerLatchMismatchRollsBack(t *testing.T) {
	expander := &fakePowerIOExpander{mismatchPin: m5ioe1.Pin3}
	power := newPowerDevice(&fakePowerPMIC{}, expander, &fakePowerPin{}, &fakePowerDelay{})
	if err := power.EnableDisplayAndTouch(); err != ErrPowerLatch {
		t.Fatalf("EnableDisplayAndTouch() error = %v, want %v", err, ErrPowerLatch)
	}
}

func TestDisplayResetRequiresPowerAndVerifiesReleasedLatch(t *testing.T) {
	expander := &fakePowerIOExpander{}
	delay := &fakePowerDelay{}
	power := newPowerDevice(&fakePowerPMIC{}, expander, &fakePowerPin{}, delay)
	if err := power.ResetDisplay(); err != ErrPowerInactive {
		t.Fatalf("inactive ResetDisplay() error = %v", err)
	}
	if err := power.EnableDisplayAndTouch(); err != nil {
		t.Fatal(err)
	}
	expander.actions = nil
	delay.delays = nil
	if err := power.ResetDisplay(); err != nil {
		t.Fatal(err)
	}
	want := []powerAction{
		{operation: 's', pin: m5ioe1.Pin5},
		{operation: 's', pin: m5ioe1.Pin5, level: true},
		{operation: 'r', pin: m5ioe1.Pin5},
	}
	assertPowerActions(t, expander.actions, want)
	if len(delay.delays) != 2 || delay.delays[0] != 10 || delay.delays[1] != 10 {
		t.Fatalf("reset delays = %v", delay.delays)
	}
}

func assertPowerActions(t *testing.T, got, want []powerAction) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("actions = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("action %d = %+v, want %+v", index, got[index], want[index])
		}
	}
}
