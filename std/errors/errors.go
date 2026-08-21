//go:build !renvo

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
func (e errorString) Error() string { return e.s }

type unwrapper interface{ Unwrap() error }
type multiUnwrapper interface{ Unwrap() []error }

func Unwrap(err error) error {
	if value, ok := err.(unwrapper); ok {
		return value.Unwrap()
	}
	return nil
}

func Is(err, target error) bool {
	if target == nil {
		return err == nil
	}
	for err != nil {
		if err == target {
			return true
		}
		if value, ok := err.(interface{ Is(error) bool }); ok && value.Is(target) {
			return true
		}
		switch value := err.(type) {
		case unwrapper:
			err = value.Unwrap()
			continue
		case multiUnwrapper:
			for _, child := range value.Unwrap() {
				if Is(child, target) {
					return true
				}
			}
		}
		return false
	}
	return false
}

type aser interface{ As(any) bool }

func As(err error, target any) bool {
	if target == nil {
		panic("errors: target cannot be nil")
	}
	for err != nil {
		if output, ok := target.(*error); ok {
			*output = err
			return true
		}
		if value, ok := err.(aser); ok && value.As(target) {
			return true
		}
		switch value := err.(type) {
		case unwrapper:
			err = value.Unwrap()
			continue
		case multiUnwrapper:
			for _, child := range value.Unwrap() {
				if As(child, target) {
					return true
				}
			}
		}
		return false
	}
	return false
}

type joinError struct{ errs []error }

func (e *joinError) Error() string {
	text := ""
	for _, err := range e.errs {
		if text != "" {
			text += "\n"
		}
		text += err.Error()
	}
	return text
}
func (e *joinError) Unwrap() []error { return e.errs }

func Join(errs ...error) error {
	count := 0
	for _, err := range errs {
		if err != nil {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	joined := &joinError{errs: make([]error, 0, count)}
	for _, err := range errs {
		if err != nil {
			joined.errs = append(joined.errs, err)
		}
	}
	return joined
}
