package context

import (
	"testing"
	"time"
)

type keyType int

func TestBackgroundTODOAndValue(t *testing.T) {
	for _, ctx := range []Context{Background(), TODO()} {
		if ctx.Done() != nil || ctx.Err() != nil {
			t.Fatal("empty context canceled")
		}
		if _, ok := ctx.Deadline(); ok {
			t.Fatal("empty context deadline")
		}
	}
	ctx := WithValue(WithValue(Background(), keyType(1), "outer"), keyType(1), "inner")
	if ctx.Value(keyType(1)) != "inner" || ctx.Value(keyType(2)) != nil {
		t.Fatal("value lookup")
	}
}

func TestCancelPropagationAndCause(t *testing.T) {
	parent, cancelParent := WithCancelCause(Background())
	child, cancelChild := WithCancel(parent)
	grandchild, _ := WithCancel(child)
	cause := errorsForTest("test cause")
	cancelParent(cause)
	for _, ctx := range []Context{parent, child, grandchild} {
		if ctx.Err() != Canceled || Cause(ctx) != cause {
			t.Fatalf("cancel state err=%v cause=%v", ctx.Err(), Cause(ctx))
		}
		select {
		case <-ctx.Done():
		default:
			t.Fatal("Done not closed")
		}
	}
	cancelChild()
	if Cause(child) != cause {
		t.Fatal("second cancel replaced cause")
	}
}

type errorsForTest string

func (e errorsForTest) Error() string { return string(e) }

func TestChildCancelDoesNotCancelParent(t *testing.T) {
	parent, cancelParent := WithCancel(Background())
	child, cancelChild := WithCancel(parent)
	cancelChild()
	if child.Err() != Canceled || parent.Err() != nil {
		t.Fatal("child cancellation propagated upward")
	}
	cancelParent()
}

func TestDeadlineExpirationInheritanceAndManualCancel(t *testing.T) {
	parentDeadline := time.Now().Add(200 * time.Millisecond)
	parent, cancelParent := WithDeadline(Background(), parentDeadline)
	defer cancelParent()

	childDeadline := time.Now().Add(20 * time.Millisecond)
	child, cancelChild := WithDeadline(parent, childDeadline)
	defer cancelChild()
	got, ok := child.Deadline()
	if !ok || !got.Equal(childDeadline) {
		t.Fatalf("child deadline = %v/%v, want %v", got, ok, childDeadline)
	}
	select {
	case <-child.Done():
	case <-time.After(time.Second):
		t.Fatal("deadline did not expire")
	}
	if child.Err() != DeadlineExceeded || Cause(child) != DeadlineExceeded {
		t.Fatalf("deadline state err=%v cause=%v", child.Err(), Cause(child))
	}

	inherited, cancelInherited := WithDeadline(parent, parentDeadline.Add(time.Second))
	defer cancelInherited()
	got, ok = inherited.Deadline()
	if !ok || !got.Equal(parentDeadline) {
		t.Fatalf("inherited deadline = %v/%v, want %v", got, ok, parentDeadline)
	}

	manual, cancelManual := WithTimeout(Background(), time.Hour)
	cancelManual()
	if manual.Err() != Canceled || Cause(manual) != Canceled {
		t.Fatalf("manual timeout cancellation err=%v cause=%v", manual.Err(), Cause(manual))
	}
}
