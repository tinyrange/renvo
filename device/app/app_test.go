package app

import (
	"reflect"
	"testing"
)

type recorder struct {
	id    int
	calls *[]int
}

func (component *recorder) Setup() { *component.calls = append(*component.calls, component.id) }
func (component *recorder) Loop()  { *component.calls = append(*component.calls, component.id+10) }

func TestSetupAndLoopPreserveComponentOrder(t *testing.T) {
	var calls []int
	components := []Component{
		&recorder{id: 1, calls: &calls},
		&recorder{id: 2, calls: &calls},
	}
	Setup(components)
	Loop(components)
	if !reflect.DeepEqual(calls, []int{1, 2, 11, 12}) {
		t.Fatalf("calls = %#v", calls)
	}
}
