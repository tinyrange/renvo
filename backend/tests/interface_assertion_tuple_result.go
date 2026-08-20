package main

type tupleResultMarshaler interface {
	MarshalValue() ([]byte, error)
}

type tupleResultValue struct{}

func (tupleResultValue) MarshalValue() ([]byte, error) {
	return []byte("PASS\n"), nil
}

func tupleResultAssertion(value any) bool {
	marshaler, ok := value.(tupleResultMarshaler)
	if !ok {
		return false
	}
	data, err := marshaler.MarshalValue()
	return err == nil && string(data) == "PASS\n"
}

func appMain() int {
	if !tupleResultAssertion(tupleResultValue{}) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
