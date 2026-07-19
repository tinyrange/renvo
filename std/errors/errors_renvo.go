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
	return err == target
}
