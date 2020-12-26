package lxdops

import (
	"testing"
)

func TestSubstitute(t *testing.T) {
	properties := make(map[string]string)
	properties["host"] = "z/host"
	properties[".container"] = "a"
	s, err := Substitute("{host}/{.container}", properties)
	if err != nil {
		t.Errorf("%v", err)
	}
	if s != "z/host/a" {
		t.Errorf(s)
	}
}

func TestSubstitute2(t *testing.T) {
	properties := make(map[string]string)
	properties["host"] = "z/host"
	_, err := Substitute("{host}/{.container}", properties)
	if err == nil {
		t.Errorf("should have caught missing property")
	}
}
