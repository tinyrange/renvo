// Package reflect exposes metadata explicitly opted into with //renvo:reflect.
// It is a Renvo API, not an implementation of Go's reflect package.
package reflect

// Field describes a directly declared exported field, in declaration order.
// An embedded field is described once; promoted fields are not flattened.
type Field struct {
	Name string
	Tag  string
}

type Struct struct {
	Name   string
	Fields []Field
}

// Describe returns metadata for an explicitly annotated named struct, or a
// pointer to one (including a nil pointer). It does not opt in nested structs.
// The compiler replaces this implementation with the opted-in type registry.
// A non-Renvo build cannot discover annotations and reports no metadata.
func Describe(value any) (Struct, bool) {
	return Struct{}, false
}

// FieldValue reads an exported field using its index in Describe's Fields.
// A nil pointer, unavailable metadata, or invalid index returns (nil, false).
func FieldValue(value any, index int) (any, bool) {
	return nil, false
}

// SetField assigns an exported field of a non-nil pointer to an opted-in struct.
// The replacement must have exactly the field's concrete type; no conversions
// are performed. Invalid targets, indices, and types leave the value unchanged.
func SetField(value any, index int, replacement any) bool {
	return false
}

// StructCopy returns a pointer to a fresh shallow copy of an annotated struct.
// A typed nil pointer produces a zero-initialized copy. Private fields are
// preserved by copying, but are not exposed through reflection.
func StructCopy(value any) (any, bool) { return nil, false }

// Assign writes an exactly typed replacement through a registered non-nil
// pointer. A pointer to any accepts any replacement, including nil.
func Assign(target any, replacement any) bool { return false }

// Collection is a snapshot of a registered slice, array, map, or pointer.
// Keys and Values correspond by index for maps; other kinds have no keys.
// Element holds a typed zero element, including for an empty collection.
// Key holds a typed zero map key and is nil for other collection kinds.
type Collection struct {
	Kind    string
	Nil     bool
	Keys    []any
	Values  []any
	Element any
	Key     any
}

// InspectCollection exposes collection types reachable through exported fields
// of annotated structs. It does not expose fields of unannotated element types.
// Built-in scalars, []any, map[string]any, []byte, and pointers to registered
// non-pointer types are also registered without a struct annotation.
func InspectCollection(value any) (Collection, bool) {
	return Collection{}, false
}

// RebuildCollection constructs a new collection with the prototype's exact
// type. Every key and element must have the exact required concrete type.
// Invalid input returns (nil, false), without modifying the prototype.
func RebuildCollection(prototype any, keys []any, values []any, nilValue bool) (any, bool) {
	return nil, false
}

// Scalar returns the underlying built-in value of a registered scalar type.
// It preserves width and signedness; it does not perform numeric coercion.
func Scalar(value any) (any, bool) { return nil, false }

// ScalarLike restores a registered named scalar type from its exact underlying
// built-in type. A mismatched replacement returns (nil, false).
func ScalarLike(prototype any, replacement any) (any, bool) { return nil, false }
