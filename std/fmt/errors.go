package fmt

type formattedError string

func (e formattedError) Error() string { return string(e) }

// Errorf formats according to a format specifier and returns the result as an
// error value. Error wrapping through %w is not part of the compact formatter.
func Errorf(format string, a ...interface{}) error {
	return formattedError(Sprintf(format, a...))
}
