package c11

import (
	"bytes"
	"testing"
)

func TestTranslateExtendedFloatingHeaderDeclarations(t *testing.T) {
	source := []byte(`typedef _Complex float complex128 __attribute__((__mode__(__TC__)));
typedef __float128 float128;
typedef int register_t __attribute__((__mode__(__word__)));
_Static_assert(sizeof(register_t) == sizeof(void *), "register width");
_Static_assert(sizeof(complex128) == 32, "complex width");
_Static_assert(_Alignof(complex128) == 16, "complex alignment");
_Static_assert(sizeof(float128) == 16, "float width");
extern void *aligned_alloc(unsigned long alignment, unsigned long size) __attribute__((__alloc_align__(1),__alloc_size__(2)));
__attribute__((deprecated("Since " "3.0"))) int old_value(void);
int value(void) { return 42; }
`)
	result := TranslateObject("main", source, nil)
	if !result.Ok || !bytes.Contains(result.Source, []byte("return 42")) {
		t.Fatalf("declarations: error=%d at=%d source=%s", result.Error, result.ErrorAt, result.Source)
	}
	for _, use := range []string{`complex128 value;`, `float128 value;`} {
		result = TranslateObject("main", append(append([]byte(nil), source...), []byte(use)...), nil)
		if result.Ok {
			t.Fatalf("unsupported extended floating value accepted: %s", use)
		}
	}
}

func TestTranslateWordMachineModeUsesTargetWidth(t *testing.T) {
	source := []byte(`typedef int signed_word __attribute__((mode(word)));
typedef __attribute__((mode(word))) unsigned int unsigned_word;
_Static_assert(sizeof(signed_word) == sizeof(void *), "signed width");
_Static_assert(sizeof(unsigned_word) == sizeof(void *), "unsigned width");
unsigned_word identity(unsigned_word value) { return value; }
`)
	for _, model := range []int{DataModelLP64, DataModelILP32, DataModelLLP64} {
		result := TranslateObjectForDataModel("main", source, nil, model)
		if !result.Ok {
			t.Fatalf("model %d: error=%d at=%d source=%s", model, result.Error, result.ErrorAt, result.Source)
		}
	}
}
