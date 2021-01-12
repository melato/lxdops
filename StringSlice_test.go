package lxdops

import (
	"testing"
)

func TestEqualArrays(t *testing.T) {
	a := []string{"a", "b"}
	if !StringSlice(a).Equals(a) {
		t.Fail()
	}
}

func TestStringSliceDiff(t *testing.T) {
	a := []string{"a", "b"}
	b := []string{"b", "c"}
	ab := StringSlice(a).Diff(b)
	if len(ab) != 1 || ab[0] != "a" {
		t.Fail()
	}
}
