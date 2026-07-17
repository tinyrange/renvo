//go:build !rtg

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
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
