//go:build !renvo

package engine

import "testing"

func TestCodePageProofSharesOnlyExactPageAndContext(t *testing.T) {
	e, c, m := prefillFixture(t)
	page := m.versions[0]
	first := e.blocks[0]
	second := *first
	second.context = CodeStamp{}
	if !e.validVersions(c, first, m.stamp) || !e.validVersions(c, &second, m.stamp) || m.reads != 1 {
		t.Fatal("same validated page/context did not share a resolver proof", m.reads)
	}
	// A different full page deliberately collides in the bounded cache. Neither
	// low 32-bit equality nor matching hash slots may substitute for its tag.
	const alias = uint64(1) << 32
	other := CodeStamp{Identity: new(CodeIdentity), Epoch: 9}
	m.versions[alias] = other
	second.dependencies = []codeDependency{{alias, other}}
	if !e.validVersions(c, &second, m.stamp) || m.reads != 2 {
		t.Fatal("full page alias skipped execute validation", m.reads)
	}
	if !e.validVersions(c, first, m.stamp) || m.reads != 3 {
		t.Fatal("collision retained another page's proof", m.reads)
	}
	// A permission/mapping/content mutation changes the complete context before
	// reentry, even when a different block last warmed this page.
	m.stamp.Epoch++
	m.versions[0] = other
	if e.validVersions(c, first, m.stamp) || m.reads != 4 {
		t.Fatal("changed mapping reused a stale dependency", m.reads)
	}
	// Failed execute permission is never remembered as a successful proof.
	m.stamp.Epoch++
	delete(m.versions, 0)
	if e.validVersions(c, first, m.stamp) {
		t.Fatal("revoked execute permission admitted")
	}
	m.versions[0] = page
	if !e.validVersions(c, first, m.stamp) || m.reads != 6 {
		t.Fatal("failed lookup was cached as authority/absence", m.reads)
	}
}

func TestCodePageProofDoesNotRememberResolverMappingTransitions(t *testing.T) {
	e, c, m := prefillFixture(t)
	initial := m.stamp
	m.transition = true
	if e.validVersions(c, e.blocks[0], initial) {
		t.Fatal("resolver's new mapping accepted under its old context")
	}
	m.transition = false
	if !e.validVersions(c, e.blocks[0], m.stamp) || m.reads != 2 {
		t.Fatal("resolver transition left a reusable stale proof", m.reads)
	}
	// No global context means no shared proof: every execute query is required.
	for i := 0; i < 2; i++ {
		if !e.validVersions(c, e.blocks[0], CodeStamp{}) {
			t.Fatal("ordinary uncached version query failed")
		}
	}
	if m.reads != 4 {
		t.Fatal("unversioned context gained a shared shortcut", m.reads)
	}
}
