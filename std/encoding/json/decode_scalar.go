package json

import (
	"errors"
	"strconv"
)

// decodeScalar converts a parsed token without an intermediate float64 for
// integers. ScalarLike restores a declared named type after range checking.
func decodeScalar(node jsonValue, prototype any) (any, error) {
	if node.kind == '0' {
		return prototype, nil
	}
	scalar, ok := scalarValue(prototype)
	if !ok {
		return nil, errors.New("json: unavailable scalar metadata")
	}
	var value any
	var err error
	switch scalar.(type) {
	case string:
		if node.kind != 's' {
			return nil, errors.New("json: expected string")
		}
		value = node.text
	case bool:
		if node.kind != 't' && node.kind != 'f' {
			return nil, errors.New("json: expected boolean")
		}
		value = node.kind == 't'
	default:
		if node.kind != 'n' {
			return nil, errors.New("json: expected number")
		}
		bits := 64
		switch scalar.(type) {
		case int, uint, uintptr:
			bits = strconv.IntSize
		case int8, uint8:
			bits = 8
		case int16, uint16:
			bits = 16
		case int32, uint32, float32:
			bits = 32
		}
		switch scalar.(type) {
		case int, int8, int16, int32, int64:
			var n int64
			n, err = strconv.ParseInt(node.text, 10, bits)
			switch scalar.(type) {
			case int:
				value = int(n)
			case int8:
				value = int8(n)
			case int16:
				value = int16(n)
			case int32:
				value = int32(n)
			case int64:
				value = n
			}
		case uint, uint8, uint16, uint32, uint64, uintptr:
			var n uint64
			n, err = strconv.ParseUint(node.text, 10, bits)
			switch scalar.(type) {
			case uint:
				value = uint(n)
			case uint8:
				value = uint8(n)
			case uint16:
				value = uint16(n)
			case uint32:
				value = uint32(n)
			case uint64:
				value = n
			case uintptr:
				value = uintptr(n)
			}
		case float32, float64:
			var n float64
			n, err = strconv.ParseFloat(node.text, bits)
			if bits == 32 {
				value = float32(n)
			} else {
				value = n
			}
		default:
			return nil, errors.New("json: unsupported scalar type")
		}
	}
	if err != nil {
		return nil, err
	}
	result, ok := scalarLike(prototype, value)
	if !ok {
		return nil, errors.New("json: incompatible scalar metadata")
	}
	return result, nil
}
