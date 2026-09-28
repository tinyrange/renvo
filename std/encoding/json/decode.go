package json

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
)

// Unmarshal decodes a complete JSON document into an addressable target.
// Named structs must explicitly opt into Renvo reflection.
func Unmarshal(data []byte, target any) error {
	node, end, err := parseJSON(data)
	if err != nil {
		return err
	}
	for end < len(data) && jsonSpace(data[end]) {
		end++
	}
	if end != len(data) {
		return &SyntaxError{Offset: int64(end + 1), message: "invalid character after top-level value"}
	}
	return decodeTarget(node, target, false)
}

func jsonSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

func decodeTarget(node jsonValue, target any, strict bool) error {
	meta, ok := inspectCollection(target)
	if !ok || meta.Kind != "pointer" || meta.Nil {
		return errors.New("json: target must be a non-nil registered pointer")
	}
	prototype := meta.Values[0]
	if meta.Element == nil {
		// An empty-interface destination takes the generic JSON representation.
		pointer, ok := inspectCollection(prototype)
		if !ok || pointer.Kind != "pointer" || pointer.Nil || node.kind == '0' {
			prototype = nil
		}
	}
	value, err := decodeValue(node, prototype, strict)
	if err != nil {
		return err
	}
	if !assignValue(target, value) {
		return errors.New("json: incompatible target metadata")
	}
	return nil
}

func decodeValue(node jsonValue, prototype any, strict bool) (any, error) {
	if prototype == nil {
		return decodeDynamic(node)
	}
	if _, ok := scalarValue(prototype); ok {
		return decodeScalar(node, prototype)
	}
	meta, collection := inspectCollection(prototype)
	if collection {
		if node.kind == '0' {
			if meta.Kind == "array" {
				return prototype, nil
			}
			return decodedCollection(prototype, nil, nil, true)
		}
		if meta.Kind == "pointer" {
			element := meta.Element
			if !meta.Nil && meta.Element != nil {
				element = meta.Values[0]
			}
			value, err := decodeValue(node, element, strict)
			if err != nil {
				return nil, err
			}
			if !meta.Nil {
				if !assignValue(prototype, value) {
					return nil, errors.New("json: incompatible pointer metadata")
				}
				return prototype, nil
			}
			return decodedCollection(prototype, nil, []any{value}, false)
		}
		if meta.Kind == "map" {
			return decodeMap(node, prototype, meta, strict)
		}
		if meta.Kind == "slice" || meta.Kind == "array" {
			if _, bytes := prototype.([]byte); bytes && node.kind == 's' {
				return base64.StdEncoding.DecodeString(node.text)
			}
			if node.kind != 'a' {
				return nil, errors.New("json: expected array")
			}
			length := len(node.items)
			if meta.Kind == "array" {
				length = len(meta.Values)
			}
			values := make([]any, length)
			for i := 0; i < length; i++ {
				values[i] = meta.Element
				if i < len(node.items) {
					element := meta.Element
					if i < len(meta.Values) && element != nil {
						element = meta.Values[i]
					}
					value, err := decodeValue(node.items[i], element, strict)
					if err != nil {
						return nil, err
					}
					values[i] = value
				}
			}
			return decodedCollection(prototype, nil, values, false)
		}
	}
	fields, ok := describeFields(prototype)
	if !ok {
		return nil, errors.New("json: unavailable struct metadata")
	}
	if node.kind == '0' {
		return prototype, nil
	}
	if node.kind != 'o' {
		return nil, errors.New("json: expected object")
	}
	working, ok := copyStruct(prototype)
	if !ok {
		return nil, errors.New("json: unavailable struct construction")
	}
	copyMeta, ok := inspectCollection(working)
	if !ok {
		return nil, errors.New("json: unavailable struct pointer metadata")
	}
	zero := copyMeta.Element
	for i, key := range node.keys {
		index := -1
		options := ""
		for j, field := range fields {
			name, opts, skip := jsonFieldTag(field.Tag)
			if skip {
				continue
			}
			if name == "" {
				name = field.Name
			}
			if name == key {
				index = j
				options = opts
				break
			}
		}
		if index < 0 {
			for j, field := range fields {
				name, opts, skip := jsonFieldTag(field.Tag)
				if skip {
					continue
				}
				if name == "" {
					name = field.Name
				}
				if strings.EqualFold(name, key) {
					index = j
					options = opts
					break
				}
			}
		}
		if index < 0 {
			if strict {
				return nil, errors.New("json: unknown field " + strconv.Quote(key))
			}
			continue
		}
		element, ok := readField(working, index)
		if !ok {
			return nil, errors.New("json: unavailable field metadata")
		}
		fieldZero, ok := readField(zero, index)
		if !ok {
			return nil, errors.New("json: unavailable zero field metadata")
		}
		if fieldZero == nil {
			pointer, ok := inspectCollection(element)
			if !ok || pointer.Kind != "pointer" || pointer.Nil || node.items[i].kind == '0' {
				element = nil
			}
		}
		item := node.items[i]
		if tagOption(options, "string") && quoteable(fieldZero) && item.kind != '0' {
			if item.kind != 's' {
				return nil, errors.New("json: invalid use of string option")
			}
			var end int
			var err error
			item, end, err = parseJSON([]byte(item.text))
			if err != nil || end != len(node.items[i].text) {
				return nil, errors.New("json: invalid quoted value")
			}
		}
		value, err := decodeValue(item, element, strict)
		if err != nil {
			return nil, err
		}
		if !writeField(working, index, value) {
			return nil, errors.New("json: incompatible field metadata")
		}
	}
	result, ok := inspectCollection(working)
	if !ok || len(result.Values) != 1 {
		return nil, errors.New("json: unavailable constructed value")
	}
	return result.Values[0], nil
}

func decodedCollection(prototype any, keys, values []any, nilValue bool) (any, error) {
	value, ok := rebuildCollection(prototype, keys, values, nilValue)
	if !ok {
		return nil, errors.New("json: incompatible collection metadata")
	}
	return value, nil
}

func decodeMap(node jsonValue, prototype any, meta collectionInfo, strict bool) (any, error) {
	if node.kind != 'o' {
		return nil, errors.New("json: expected object")
	}
	keys := append([]any{}, meta.Keys...)
	values := append([]any{}, meta.Values...)
	for i, name := range node.keys {
		keyNode := jsonValue{kind: 'n', text: name}
		keyScalar, ok := scalarValue(meta.Key)
		if !ok {
			return nil, errors.New("json: unsupported map key")
		}
		switch keyScalar.(type) {
		case string:
			keyNode.kind = 's'
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr:
		default:
			return nil, errors.New("json: unsupported map key")
		}
		key, err := decodeScalar(keyNode, meta.Key)
		if err != nil {
			return nil, err
		}
		value, err := decodeValue(node.items[i], meta.Element, strict)
		if err != nil {
			return nil, err
		}
		index := -1
		for j, old := range keys {
			if old == key {
				index = j
				break
			}
		}
		if index < 0 {
			keys = append(keys, key)
			values = append(values, value)
		} else {
			values[index] = value
		}
	}
	return decodedCollection(prototype, keys, values, false)
}

func decodeDynamic(node jsonValue) (any, error) {
	switch node.kind {
	case '0':
		return nil, nil
	case 't':
		return true, nil
	case 'f':
		return false, nil
	case 's':
		return node.text, nil
	case 'n':
		return strconv.ParseFloat(node.text, 64)
	case 'a':
		out := make([]any, len(node.items))
		for i, item := range node.items {
			value, err := decodeDynamic(item)
			if err != nil {
				return nil, err
			}
			out[i] = value
		}
		return out, nil
	case 'o':
		out := make(map[string]any)
		for i, key := range node.keys {
			value, err := decodeDynamic(node.items[i])
			if err != nil {
				return nil, err
			}
			out[key] = value
		}
		return out, nil
	}
	return nil, errors.New("json: invalid parsed value")
}
