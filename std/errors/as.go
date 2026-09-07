package errors

// As finds the first matching error in a depth-first traversal of err's tree.
// The target must be a non-nil pointer to an interface or an error type.
func As(err error, target any) bool {
	if err == nil {
		return false
	}
	for err != nil {
		valid, matched := asTarget(err, target)
		if !valid {
			panic("errors: target must be a non-nil pointer to an interface or error type")
		}
		if matched {
			return true
		}
		if custom, ok := err.(interface{ As(any) bool }); ok && custom.As(target) {
			return true
		}
		if single, ok := err.(interface{ Unwrap() error }); ok {
			err = single.Unwrap()
			continue
		}
		if multiple, ok := err.(interface{ Unwrap() []error }); ok {
			for _, child := range multiple.Unwrap() {
				if As(child, target) {
					return true
				}
			}
		}
		return false
	}
	return false
}
