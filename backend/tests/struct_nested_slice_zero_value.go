package main

type Inner struct {
	Values []int
	Bytes  []byte
}
type Outer struct {
	Before int
	Nested Inner
	Items  []Inner
	After  int
}

func checkZero() bool {
	var value Outer
	if value.Before != 0 || value.After != 0 || value.Nested.Values != nil || value.Nested.Bytes != nil || value.Items != nil {
		return false
	}
	if len(value.Nested.Values) != 0 || cap(value.Nested.Values) != 0 || len(value.Items) != 0 || cap(value.Items) != 0 {
		return false
	}
	value.Nested.Values = append(value.Nested.Values, 42)
	value.Items = append(value.Items, value.Nested)
	return value.Items[0].Values[0] == 42
}

func appMain(args []string) int {
	if !checkZero() || !checkZero() {
		panic("nested slice zero value")
	}
	print("PASS\n")
	return 0
}
