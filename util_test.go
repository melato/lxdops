package lxdops

import (
	"testing"
)

func TestEqualArrays(t *testing.T) {
	a := []string{"a", "b"}
	if !EqualArrays(a, a) {
		t.Fail()
	}
}
