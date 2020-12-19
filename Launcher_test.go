package lxdops

import (
	"testing"
)

func TestContainerName(t *testing.T) {
	name := ContainerName("a.yaml")
	if name != "a" {
		t.Errorf("name: %s expected:%s", name, "a")
	}
}
