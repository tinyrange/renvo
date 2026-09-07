package fmt

// Stringer supplies the text representation used by string formatting verbs.
type Stringer interface{ String() string }

func methodString(value interface{}) (string, bool) {
	// Errors take precedence when a value implements both interfaces.
	if err, ok := value.(error); ok {
		return err.Error(), true
	}
	if s, ok := value.(Stringer); ok {
		return s.String(), true
	}
	return "", false
}
