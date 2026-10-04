package left

type Record[T any] = struct {
	value  T
	Public int
}
type UnicodePrivate[T any] = struct{ λ T }
type UnicodePublic[T any] = struct{ É T }
type Titlecase[T any] = struct{ ǅ T }
type PrivateAlias = interface{ seal() int }
type UnicodePrivateAlias = interface{ λ() int }
type UnicodePublicAlias = interface{ É() int }

func RecordValue[T any](v T) any             { return Record[T]{value: v, Public: 17} }
func RecordMatches[T any](v any) bool        { r, ok := v.(Record[T]); return ok && r.Public == 17 }
func PrivateValue[T any](v T) any            { return UnicodePrivate[T]{λ: v} }
func PrivateMatches[T any](v any) bool       { _, ok := v.(UnicodePrivate[T]); return ok }
func PublicValue[T any](v T) any             { return UnicodePublic[T]{É: v} }
func PublicMatches[T any](v any) bool        { _, ok := v.(UnicodePublic[T]); return ok }
func TitlecaseValue[T any](v T) any          { return Titlecase[T]{ǅ: v} }
func TitlecaseMatches[T any](v any) bool     { _, ok := v.(Titlecase[T]); return ok }
func MethodMatches(v any) bool               { _, ok := v.(PrivateAlias); return ok }
func UnicodePrivateMethodMatches(v any) bool { _, ok := v.(UnicodePrivateAlias); return ok }
func UnicodePublicMethodMatches(v any) bool  { _, ok := v.(UnicodePublicAlias); return ok }

type ReaderAlias = interface{ Read() int }
type NamedReader interface{ Read() int }
type CombinedAlias = interface {
	ReaderAlias
	Write(...int) (int, bool)
}
type valueType int

func (v valueType) seal() int { return int(v) }
func (v valueType) λ() int    { return int(v) }
func (v valueType) É() int    { return int(v) }
func MethodValue() any        { return valueType(42) }
