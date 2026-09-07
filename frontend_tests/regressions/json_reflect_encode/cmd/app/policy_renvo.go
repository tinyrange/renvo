//go:build renvo

package main

import "encoding/json"

type Plain struct{ Secret int }

//renvo:reflect
type Outer struct{ Child Plain }

func checkRenvoPolicy() {
	data, err := json.Marshal(Plain{Secret: 42})
	if err == nil || data != nil {
		panic("JSON exposed unannotated struct")
	}
	data, err = json.Marshal(Outer{Child: Plain{Secret: 42}})
	if err == nil || data != nil {
		panic("JSON implicitly opted in nested struct")
	}
}
