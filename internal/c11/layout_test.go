package c11

import (
	"bytes"
	"testing"
)

func TestTranslateUsesCLayoutForAdjacentNarrowStructFields(t *testing.T) {
	result := Translate("main", []byte(`
struct hash {
	unsigned size;
	unsigned count;
	void *first;
};

unsigned count(struct hash *value) {
	return value->count;
}
`))
	if result.Error != TranslateOK {
		t.Fatalf("Translate failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("type __c_struct_hash struct{__c_align uint64;__c_tail [8]byte}")) ||
		!bytes.Contains(result.Source, []byte(".__c_ptr_4_count()")) {
		t.Fatalf("narrow fields did not use their exact C offsets:\n%s", result.Source)
	}
}

func TestTranslateCastAppliesAfterPostfixMemberAccess(t *testing.T) {
	result := Translate("main", []byte(`
typedef struct page { void *pBuf; void *pExtra; } sqlite3_pcache_page;
typedef struct header { void *pPage; void *pData; void *pExtra; } PgHdr;

PgHdr *header(sqlite3_pcache_page *pPage) {
	return (PgHdr *)pPage->pExtra;
}
`))
	if result.Error != TranslateOK {
		t.Fatalf("Translate failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if bytes.Contains(result.Source, []byte("((*__c_struct_header)(pPage)).pExtra")) ||
		!bytes.Contains(result.Source, []byte("pPage.pExtra")) {
		t.Fatalf("cast bound before postfix member access:\n%s", result.Source)
	}
}

func TestTranslatePropagatesIndirectNestedLayout(t *testing.T) {
	result := Translate("main", []byte(`
struct flags { unsigned first; unsigned second; };
struct item { void *name; struct flags flags; void *value; };

void *value(struct item *item) { return item->value; }
`))
	if result.Error != TranslateOK {
		t.Fatalf("Translate failed: error=%d at=%d", result.Error, result.ErrorAt)
	}
	if !bytes.Contains(result.Source, []byte("type __c_struct_item struct{__c_align uint64;__c_tail [16]byte}")) ||
		!bytes.Contains(result.Source, []byte(".__c_ptr_16_value()")) {
		t.Fatalf("indirect child layout did not propagate to its parent:\n%s", result.Source)
	}
}
