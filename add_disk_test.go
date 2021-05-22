package lxdops

import (
	"fmt"
	"testing"
)

func verifyDeviceName(t *testing.T, path, expected string, fn func(string) bool) {
	name := CreateDeviceName(path, fn)
	if name != expected {
		fmt.Printf("path=%s name=%s expected=%s\n", path, name, expected)
		t.Fail()
	}
}

func TestCreateDeviceName(t *testing.T) {
	verifyDeviceName(t, "/a/b", "b", func(string) bool { return false })
	verifyDeviceName(t, "/a/b", "a_b", func(s string) bool { return s == "b" })
}
