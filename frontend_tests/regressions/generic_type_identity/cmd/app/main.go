package main

import (
	"example.com/genericidentity/left"
	"example.com/genericidentity/right"
)

func Box[T any](v T) any        { return v }
func Matches[T any](v any) bool { _, ok := v.(T); return ok }
func main() {
	var reader *left.ReaderAlias
	if !Matches[*interface{ Read() int }](Box(reader)) {
		panic("anonymous interface pointer")
	}
	var slice []left.ReaderAlias
	if !Matches[[]interface{ Read() int }](Box(slice)) {
		panic("anonymous interface slice")
	}
	var array [2]left.ReaderAlias
	if !Matches[[2]interface{ Read() int }](Box(array)) {
		panic("anonymous interface array")
	}
	var record struct{ R left.ReaderAlias }
	if !Matches[struct{ R interface{ Read() int } }](Box(record)) {
		panic("anonymous interface field")
	}
	var function func(left.ReaderAlias) left.ReaderAlias
	if !Matches[func(interface{ Read() int }) interface{ Read() int }](Box(function)) {
		panic("anonymous interface signature")
	}
	var combined *left.CombinedAlias
	if !Matches[*interface {
		Write(...int) (int, bool)
		Read() int
	}](Box(combined)) {
		panic("embedded method set")
	}
	if Matches[*interface {
		Write([]int) (int, bool)
		Read() int
	}](Box(combined)) {
		panic("variadic identity")
	}
	var named *left.NamedReader
	if Matches[*interface{ Read() int }](Box(named)) {
		panic("defined interface identity")
	}
	var err *error
	if Matches[*interface{ Error() string }](Box(err)) || !Matches[*error](Box(err)) {
		panic("error identity")
	}
	if !left.RecordMatches[int](left.RecordValue(42)) || right.RecordMatches[int](left.RecordValue(42)) {
		panic("private field identity")
	}
	if !right.RecordMatches[string](right.RecordValue("value")) || left.RecordMatches[string](right.RecordValue("value")) {
		panic("private field reverse identity")
	}
	if !left.PrivateMatches[int](left.PrivateValue(42)) || right.PrivateMatches[int](left.PrivateValue(42)) {
		panic("unicode private field")
	}
	if !right.PublicMatches[int](left.PublicValue(42)) {
		panic("unicode exported field")
	}
	if right.TitlecaseMatches[int](left.TitlecaseValue(42)) {
		panic("titlecase private field")
	}
	var private *left.PrivateAlias
	if Matches[*right.PrivateAlias](Box(private)) || !Matches[*left.PrivateAlias](Box(private)) {
		panic("private interface method")
	}
	var unicodePrivate *left.UnicodePrivateAlias
	if Matches[*right.UnicodePrivateAlias](Box(unicodePrivate)) {
		panic("unicode private interface method")
	}
	var unicodePublic *left.UnicodePublicAlias
	if !Matches[*right.UnicodePublicAlias](Box(unicodePublic)) {
		panic("unicode exported interface method")
	}
	if !left.MethodMatches(left.MethodValue()) || right.MethodMatches(left.MethodValue()) {
		panic("private implementation")
	}
	if !left.UnicodePrivateMethodMatches(left.MethodValue()) || right.UnicodePrivateMethodMatches(left.MethodValue()) {
		panic("unicode private implementation")
	}
	if !right.UnicodePublicMethodMatches(left.MethodValue()) {
		panic("unicode exported implementation")
	}
	print("PASS\n")
}
