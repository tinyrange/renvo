package errors

import "testing"

type asDetail struct{ n int }

func (e *asDetail) Error() string { return "detail" }
func (e *asDetail) Number() int   { return e.n }

type asNumbered interface{ Number() int }
type asValue int

func (e asValue) Error() string { return "value" }

type asOther struct{}

func (e *asOther) Error() string { return "other" }

type asTree struct{ children []error }

func (e asTree) Error() string   { return "tree" }
func (e asTree) Unwrap() []error { return e.children }

type asCustom struct{ detail *asDetail }

func (e asCustom) Error() string { return "custom" }
func (e asCustom) As(target any) bool {
	if p, ok := target.(**asDetail); ok {
		*p = e.detail
		return true
	}
	return false
}

func TestAs(t *testing.T) {
	first := &asDetail{n: 1}
	second := &asDetail{n: 2}
	var target *asDetail
	if !As(wrappedError{inner: first}, &target) || target != first {
		t.Fatal("single unwrap")
	}
	tree := asTree{children: []error{nil, wrappedError{inner: first}, second}}
	if !As(tree, &target) || target != first {
		t.Fatal("depth first order")
	}
	if !As(asCustom{detail: second}, &target) || target != second {
		t.Fatal("custom As")
	}
	if As(&asOther{}, &target) || target != second {
		t.Fatal("nonmatch changed target")
	}
	var nilDetail *asDetail
	if !As(nilDetail, &target) || target != nil {
		t.Fatal("typed nil error")
	}
	var generic any
	if !As(first, &generic) || generic != first {
		t.Fatal("any target")
	}
	var plain error
	if !As(first, &plain) || plain != first {
		t.Fatal("error target")
	}
	var numbered asNumbered
	if !As(first, &numbered) || numbered.Number() != 1 {
		t.Fatal("interface target")
	}
	var scalar asValue
	if !As(asValue(7), &scalar) || scalar != 7 {
		t.Fatal("value error")
	}
	if As(nil, nil) {
		t.Fatal("nil error")
	}
}

func TestAsInvalidTarget(t *testing.T) {
	var nilTarget **asDetail
	var invalid int
	values := []any{nil, nilTarget, 3, &invalid}
	for _, target := range values {
		panicked := false
		func() {
			defer func() {
				if recover() != nil {
					panicked = true
				}
			}()
			As(&asDetail{}, target)
		}()
		if !panicked {
			t.Fatal("invalid target accepted")
		}
	}
}
