package lxdops

import (
	"errors"
	"fmt"
)

type ContainerOps struct {
}

func (t *ContainerOps) listProfiles(c *Container) error {
	for _, profile := range c.Profiles {
		fmt.Println(profile)
	}
	return nil
}

func (t *ContainerOps) run(args []string, f func(c *Container) error) error {
	if len(args) != 1 {
		return errors.New("usage: <container>")
	}
	c, err := ListContainer(args[0])
	if err != nil {
		return err
	}
	return f(c)
}

func (t *ContainerOps) Profiles(args []string) error {
	return t.run(args, t.listProfiles)
}
