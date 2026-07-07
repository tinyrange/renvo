package testing

import hosttesting "testing"

func TestTMethods(t *hosttesting.T) {
	var tt T
	tt.Log("hello", 7)
	if len(tt.logs) != 1 || tt.logs[0] != "hello 7" {
		t.Fatalf("Log = %#v", tt.logs)
	}
	if tt.Run("child", func(child *T) { child.Error("bad") }) || !tt.Failed() {
		t.Fatalf("Run/Failed state mismatch")
	}
}

func TestFailNowPanics(t *hosttesting.T) {
	var tt T
	defer func() {
		if recover() == nil || !tt.Failed() {
			t.Fatalf("FailNow did not panic and fail")
		}
	}()
	tt.FailNow()
}
