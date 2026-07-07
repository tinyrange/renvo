//go:build rtg

package errors

type errorString struct {
	s  string
	id int
}

var nextErrorID int

func New(text string) errorString {
	nextErrorID++
	return errorString{s: text, id: nextErrorID}
}

func (e errorString) Error() string {
	return e.s
}

func Is(err errorString, target errorString) bool {
	return err.id == target.id
}
