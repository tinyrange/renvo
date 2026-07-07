//go:build !rtg

package errors

type errorString struct {
	s string
}

func New(text string) error {
	return &errorString{s: text}
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
