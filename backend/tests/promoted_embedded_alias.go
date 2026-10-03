package main

type promotedAliasBase struct{ value int }

func (v promotedAliasBase) get() int { return v.value }

type promotedAliasOuter struct{ promotedEmbeddedAlias }
type promotedEmbeddedAlias = promotedAliasBase

func appMain(args []string) int {
	v := promotedAliasOuter{promotedAliasBase{42}}
	var receiver interface{ get() int } = v
	bound := receiver.get
	if receiver.get() != 42 || bound() != 42 {
		return 1
	}
	var dynamic interface{} = v
	if view, ok := dynamic.(interface{ get() int }); !ok || view.get() != 42 {
		return 2
	}
	print("PASS\n")
	return 0
}
