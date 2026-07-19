package errors

import "testing"

type wrappedError struct {
	inner error
}

func (w wrappedError) Error() string {
	return "wrapped"
}

func (w wrappedError) Unwrap() error {
	return w.inner
}

func TestNewAndIs(t *testing.T) {
	err := New("bad")
	if err == nil || err.Error() != "bad" {
		t.Fatalf("New error = %v", err)
	}
	if !Is(err, err) {
		t.Fatalf("Is did not match identical error")
	}
	if !Is(wrappedError{inner: err}, err) {
		t.Fatalf("Is did not unwrap")
	}
	if Is(err, New("bad")) {
		t.Fatalf("Is matched different error with same text")
	}
	if !Is(nil, nil) {
		t.Fatalf("Is(nil, nil) = false")
	}
}
