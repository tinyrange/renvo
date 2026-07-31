package main

// renvo:linkstatic example.dll,DefinitionStaticImport
func definitionStaticImport(value int) {
}

func appMain(args []string) int {
	definitionStaticImport(7)
	print("PASS\n")
	return 0
}
