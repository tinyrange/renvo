package errors

import "testing"

type wrappedError struct{ inner error }

func (w wrappedError) Error() string { return "wrapped" }
func (w wrappedError) Unwrap() error { return w.inner }

type matchingError struct{ wanted error }

func (matchingError) Error() string          { return "matching" }
func (m matchingError) Is(target error) bool { return target == m.wanted }

type specialError struct{ text string }

func (e *specialError) Error() string { return e.text }

type asWrapper struct{ value *specialError }

func (w asWrapper) Error() string { return "as wrapper" }
func (w asWrapper) As(target any) bool {
	output, ok := target.(**specialError)
	if !ok {
		return false
	}
	*output = w.value
	return true
}

func TestNewIsAndUnwrap(t *testing.T) {
	err := New("bad")
	if err == nil || err.Error() != "bad" {
		t.Fatalf("New error = %v", err)
	}
	wrapped := wrappedError{inner: wrappedError{inner: err}}
	if !Is(wrapped, err) || Unwrap(wrapped) == nil {
		t.Fatal("Is/Unwrap did not traverse")
	}
	if Is(err, New("bad")) {
		t.Fatal("Is matched different error with same text")
	}
	if !Is(nil, nil) {
		t.Fatal("Is(nil, nil) = false")
	}
	if !Is(matchingError{wanted: err}, err) {
		t.Fatal("custom Is was not called")
	}
}

func TestAs(t *testing.T) {
	wanted := &specialError{text: "special"}
	var found *specialError
	if !As(wrappedError{inner: asWrapper{value: wanted}}, &found) || found != wanted {
		t.Fatal("custom As did not assign")
	}
	var generic error
	if !As(wanted, &generic) || generic != wanted {
		t.Fatal("As to *error did not assign")
	}
}

func TestJoin(t *testing.T) {
	first, second := New("first"), New("second")
	joined := Join(nil, first, second)
	if joined == nil || joined.Error() != "first\nsecond" {
		t.Fatalf("Join text = %v", joined)
	}
	if !Is(joined, first) || !Is(joined, second) {
		t.Fatal("Is did not traverse Join")
	}
	if Join(nil, nil) != nil {
		t.Fatal("nil Join was non-nil")
	}
	multi, ok := joined.(interface{ Unwrap() []error })
	if !ok || len(multi.Unwrap()) != 2 {
		t.Fatal("Join did not expose children")
	}
}
