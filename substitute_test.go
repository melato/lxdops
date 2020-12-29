package lxdops

import (
	"errors"
	"testing"
)

func SubstituteTestMapFunc(properties map[string]string) func(key string) (string, error) {
	return func(key string) (string, error) {
		value, found := properties[key]
		if found {
			return value, nil
		}
		return "", errors.New("property not found: " + key)
	}
}

func TestSubstitute(t *testing.T) {
	properties := make(map[string]string)
	properties["host"] = "z/host"
	properties[".container"] = "a"
	s, err := Substitute("(host)/(.container)", SubstituteTestMapFunc(properties))
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
	_, err := Substitute("(host)/(.container)", SubstituteTestMapFunc(properties))
	if err == nil {
		t.Errorf("should have caught missing property")
	}
}

func TestSubstitute0(t *testing.T) {
	s, err := Substitute("", func(key string) (string, error) { return "", errors.New("x") })
	if err != nil {
		t.Fail()
	}
	if s != "" {
		t.Errorf("expected empty result")
	}
}
