package asmtext

import "testing"

func TestFunctionDiscoveryUsesDefinitionOrder(t *testing.T) {
	source := []byte(".global second,first\nfirst: ret\n/* ignore: fake */\nsecond: ret\n")
	lines, err := Scan(source)
	if err.Message != "" {
		t.Fatal(err)
	}
	functions, err := Functions(lines)
	if err.Message != "" || len(functions) != 2 || functions[0].Name != "first" || functions[1].Name != "second" || functions[0].Offset >= functions[1].Offset {
		t.Fatalf("discovery: %+v %+v", functions, err)
	}
	for _, bad := range []string{".globl missing", ".globl x\nx: ret\nx: ret", ".globl x,x\nx: ret"} {
		lines, err = Scan([]byte(bad))
		if err.Message != "" {
			t.Fatal(err)
		}
		if _, err = Functions(lines); err.Message == "" {
			t.Fatal("accepted", bad)
		}
	}
}
