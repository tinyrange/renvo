package main

type aliasSignatureValue struct {
	n int
}

type aliasSignatureAlias = aliasSignatureValue

func newAliasSignatureValue() *aliasSignatureAlias {
	return &aliasSignatureValue{n: 1}
}

func (value *aliasSignatureValue) resetAliasSignatureValue() {
	value.n = 0
}

// A later signature reference must not split the alias from its receiver type.
type aliasSignatureHolder struct {
	value *aliasSignatureAlias
}

func appMain(args []string) int {
	value := newAliasSignatureValue()
	value.resetAliasSignatureValue()
	if value.n != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
