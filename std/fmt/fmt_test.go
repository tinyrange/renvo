package fmt

import "testing"

func TestPrintlnSpacing(t *testing.T) {
	if got := Sprintln("a", 1, "b"); got != "a 1 b\n" {
		t.Fatalf("Sprintln = %q", got)
	}
	if got := Sprintln(); got != "\n" {
		t.Fatalf("empty Sprintln = %q", got)
	}
	var out sink
	n, err := Fprintln(&out, "a", 1, "b")
	if n != 6 || err != nil || string(out.data) != "a 1 b\n" {
		t.Fatalf("Fprintln = %d, %v, %q", n, err, out.data)
	}
}

func TestErrorf(t *testing.T) {
	err := Errorf("failed %s: %d", "item", 7)
	if got := err.Error(); got != "failed item: 7" {
		t.Fatalf("Errorf = %q", got)
	}
}

type sink struct {
	data []byte
}

func (s *sink) Write(p []byte) (int, error) {
	s.data = append(s.data, p...)
	return len(p), nil
}

func TestSprintfAndSprint(t *testing.T) {
	if Sprint("a", "b") != "ab" || Sprint(1, 2, "x") != "1 2x" {
		t.Fatalf("Sprint failed")
	}
	got := Sprintf("%s:%d:%x:%x:%q:%%:%v", "id", -7, 255, -15, "go\n", true)
	if got != "id:-7:ff:-f:\"go\\n\":%:true" {
		t.Fatalf("Sprintf = %q", got)
	}
	got = Sprintf("%d:%d:%d:%d:%x:%x", int8(-1), int16(-2), uint8(3), uint16(4), int32(15), uint32(16))
	if got != "-1:-2:3:4:f:10" {
		t.Fatalf("Sprintf integer widths = %q", got)
	}
}

func TestFprint(t *testing.T) {
	var s sink
	n, err := Fprintf(&s, "%s=%d", "value", 12)
	if err != nil || n != 8 || string(s.data) != "value=12" {
		t.Fatalf("Fprintf = %d %v %q", n, err, string(s.data))
	}
	n, err = Fprintln(&s, " ok")
	if err != nil || n != 4 || string(s.data) != "value=12 ok\n" {
		t.Fatalf("Fprintln = %d %v %q", n, err, string(s.data))
	}
}
