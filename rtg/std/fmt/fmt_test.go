package fmt

import "testing"

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
