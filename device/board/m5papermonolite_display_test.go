//go:build m5papermonolite

package board

import (
	"errors"
	"testing"

	"renvo.dev/device/display/ssd1677"
)

type fakeDisplayTransport struct {
	initializeCalls int
	deactivateCalls int
	err             error
}

func (transport *fakeDisplayTransport) initialize() error {
	transport.initializeCalls++
	return transport.err
}

func (transport *fakeDisplayTransport) deactivate() { transport.deactivateCalls++ }

type fakeDisplayProtocol struct {
	fullCalls       int
	partialCalls    int
	fastCalls       int
	grayCalls       int
	invalidateCalls int
	err             error
}

func (protocol *fakeDisplayProtocol) FullMonochrome([]byte) error {
	protocol.fullCalls++
	return protocol.err
}

func (protocol *fakeDisplayProtocol) PartialMonochrome([]byte) error {
	protocol.partialCalls++
	return protocol.err
}

func (protocol *fakeDisplayProtocol) FastMonochrome([]byte, ssd1677.RefreshPoller) error {
	protocol.fastCalls++
	return protocol.err
}

func (protocol *fakeDisplayProtocol) FullGray([]byte, []byte) error {
	protocol.grayCalls++
	return protocol.err
}

func (protocol *fakeDisplayProtocol) FullGrayStream(ssd1677.GrayPlaneSource) error {
	protocol.grayCalls++
	return protocol.err
}

func (protocol *fakeDisplayProtocol) InvalidateBaseline() { protocol.invalidateCalls++ }

type fakeDisplayPower struct {
	enableCalls  int
	disableCalls int
	enableErr    error
}

func (power *fakeDisplayPower) EnableDisplayAndTouch() error {
	power.enableCalls++
	return power.enableErr
}

func (power *fakeDisplayPower) DisableDisplayAndTouch() error {
	power.disableCalls++
	return nil
}

func TestPaperDisplayEnablesOnceAndShutsDown(t *testing.T) {
	transport := &fakeDisplayTransport{}
	protocol := &fakeDisplayProtocol{}
	power := &fakeDisplayPower{}
	display := newPaperDisplay(transport, protocol, power)
	if err := display.FullMonochrome(nil); err != nil {
		t.Fatal(err)
	}
	if err := display.PartialMonochrome(nil); err != nil {
		t.Fatal(err)
	}
	if err := display.FastMonochrome(nil, nil); err != nil {
		t.Fatal(err)
	}
	if power.enableCalls != 1 || transport.initializeCalls != 1 || protocol.fullCalls != 1 || protocol.partialCalls != 1 || protocol.fastCalls != 1 {
		t.Fatalf("enable=%d init=%d full=%d partial=%d fast=%d", power.enableCalls, transport.initializeCalls, protocol.fullCalls, protocol.partialCalls, protocol.fastCalls)
	}
	if err := display.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if power.disableCalls != 1 || transport.deactivateCalls != 1 || protocol.invalidateCalls != 1 || display.active {
		t.Fatalf("disable=%d deactivate=%d invalidate=%d active=%v", power.disableCalls, transport.deactivateCalls, protocol.invalidateCalls, display.active)
	}
}

func TestPaperDisplayTransportFailureRollsBackPower(t *testing.T) {
	transport := &fakeDisplayTransport{err: errors.New("injected SPI setup failure")}
	power := &fakeDisplayPower{}
	display := newPaperDisplay(transport, &fakeDisplayProtocol{}, power)
	if err := display.Enable(); err == nil {
		t.Fatal("Enable succeeded after transport failure")
	}
	if power.enableCalls != 1 || power.disableCalls != 1 || display.active {
		t.Fatalf("enable=%d disable=%d active=%v", power.enableCalls, power.disableCalls, display.active)
	}
}

func TestPaperDisplayProtocolFailurePowersDownAndInvalidates(t *testing.T) {
	injected := errors.New("injected protocol failure")
	transport := &fakeDisplayTransport{}
	protocol := &fakeDisplayProtocol{err: injected}
	power := &fakeDisplayPower{}
	display := newPaperDisplay(transport, protocol, power)
	if err := display.FullGray(nil, nil); err != injected {
		t.Fatalf("FullGray error = %v", err)
	}
	if power.disableCalls != 1 || transport.deactivateCalls != 1 || protocol.invalidateCalls != 1 || display.active {
		t.Fatalf("disable=%d deactivate=%d invalidate=%d active=%v", power.disableCalls, transport.deactivateCalls, protocol.invalidateCalls, display.active)
	}
}
