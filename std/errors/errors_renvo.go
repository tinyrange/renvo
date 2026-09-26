//go:build renvo

package errors

type errorString struct {
	s  string
	id int
}

var nextErrorID int

func New(text string) error {
	nextErrorID++
	return errorString{s: text, id: nextErrorID}
}

func (e errorString) Error() string {
	return e.s
}

func Is(err error, target error) bool {
	if target == nil {
		return err == nil
	}
	for err != nil {
		if err == target {
			return true
		}
		if custom, ok := err.(interface{ Is(error) bool }); ok && custom.Is(target) {
			return true
		}
		wrapped, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = wrapped.Unwrap()
	}
	return false
}
