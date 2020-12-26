package lxdops

import (
	"testing"
)

func SubstituteTestMapFunc(properties map[string]string) func(key string) (string, bool) {
	return func(key string) (string, bool) {
		value, found := properties[key]
		return value, found
	}
}

func TestSubstitute(t *testing.T) {
	properties := make(map[string]string)
	properties["host"] = "z/host"
	properties[".container"] = "a"
	s, err := Substitute("{host}/{.container}", SubstituteTestMapFunc(properties))
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
	_, err := Substitute("{host}/{.container}", SubstituteTestMapFunc(properties))
	if err == nil {
		t.Errorf("should have caught missing property")
	}
}

func TestSubstitute0(t *testing.T) {
	s, err := Substitute("", func(key string) { return "", false })
	if err != nil {
		t.Fail()
	}
	if s != "" {
		t.Errorf("expected empty result")
	}
}
