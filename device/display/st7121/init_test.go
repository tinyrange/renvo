package st7121

import "testing"

type recordingTransport struct {
	commands int
	delay    int
}

func (transport *recordingTransport) Command(byte, []byte)   { transport.commands++ }
func (transport *recordingTransport) Delay(milliseconds int) { transport.delay += milliseconds }

func TestInitializeSendsCommandsAndDelays(t *testing.T) {
	transport := &recordingTransport{}
	Initialize(transport)
	if transport.commands == 0 || transport.delay < 120 {
		t.Fatalf("initialization commands/delay = %d/%d", transport.commands, transport.delay)
	}
}
