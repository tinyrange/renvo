package unit

import "testing"

func TestForeignProgramAndEntrypointNodes(t *testing.T) {
	base, ok := MarshalCore(CoreProgram{
		Package: "main", ImportPath: "example", Text: []byte("package main\nfunc hello(){}\n"),
		Tokens: []Token{MakeToken(TokenPackage, 0, 7, 1), MakeToken(TokenIdent, 8, 4, 1), MakeToken(TokenFunc, 13, 4, 2), MakeToken(TokenIdent, 18, 5, 2), MakeToken(TokenOp, 23, 1, 2), MakeToken(TokenOp, 24, 1, 2), MakeToken(TokenOp, 25, 1, 2), MakeToken(TokenOp, 26, 1, 2), MakeToken(TokenEOF, 28, 0, 3)},
		Funcs:  []Func{{NameStart: 18, NameEnd: 23, StartTok: 2, NameTok: 3, BodyStart: 6, BodyEnd: 7, EndTok: 8}},
	})
	if !ok {
		t.Fatal("marshal failed")
	}
	withEntry, ok := BindEntrypoint(base, 0)
	if !ok {
		t.Fatal("bind entrypoint failed")
	}
	child, ok := BindTarget(base, TargetBinding{Target: "linux/amd64", Definition: string(make([]byte, 32)), DescriptorVersion: 1})
	if !ok {
		t.Fatal("bind child target failed")
	}
	withForeign, ok := BindForeignPrograms(withEntry, []ForeignProgram{{Global: 8, Kind: ForeignProgramBytes, Target: "linux/amd64", Unit: child}})
	if !ok || len(withForeign) <= len(withEntry) || readUint32Foreign(withForeign, 10) != len(withForeign)-14 {
		t.Fatalf("foreign binding failed: ok=%v size=%d", ok, len(withForeign))
	}
	programs, ok := ReadForeignPrograms(withForeign)
	if !ok || len(programs) != 1 || programs[0].Global != 8 || programs[0].Target != "linux/amd64" || len(programs[0].Unit) == 0 {
		t.Fatalf("foreign read = %#v, %v", programs, ok)
	}
	programs[0].Unit = nil
	programs[0].Artifact = []byte("binary")
	resolved, ok := ResolveForeignPrograms(withForeign, programs)
	if !ok {
		t.Fatal("resolve foreign programs failed")
	}
	programs, ok = ReadForeignPrograms(resolved)
	if !ok || len(programs) != 1 || len(programs[0].Unit) != 0 || string(programs[0].Artifact) != "binary" {
		t.Fatalf("resolved foreign read = %#v, %v", programs, ok)
	}
	if _, ok := BindEntrypoint(withForeign, 0); ok {
		t.Fatal("duplicate entrypoint was accepted")
	}
	programs[0].Kind = ForeignProgramEntrypoint
	programs[0].EntryOffset = len(programs[0].Artifact)
	invalid, ok := ResolveForeignPrograms(resolved, programs)
	if !ok {
		t.Fatal("could not construct malformed foreign table")
	}
	if _, ok := ReadForeignPrograms(invalid); ok {
		t.Fatal("foreign table accepted an out-of-range entrypoint")
	}
}
