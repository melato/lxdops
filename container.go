package lxdops

import (
	"errors"
	"fmt"

	//"melato.org/lxdops/lxd"
	lxd "github.com/lxc/lxd/shared/api"
)

type ContainerOps struct {
}

func (t *ContainerOps) listProfiles(c *lxd.ContainerFull) error {
	for _, profile := range c.Profiles {
		fmt.Println(profile)
	}
	return nil
}

func (t *ContainerOps) printNetwork(c *lxd.ContainerFull) error {
	if c.State != nil {
		for name, net := range c.State.Network {
			for _, a := range net.Addresses {
				fmt.Printf("%s %s %s %s/%s\n", name, a.Family, a.Scope, a.Address, a.Netmask)
			}
		}
	}
	return nil
}

func (t *ContainerOps) run(args []string, f func(c *lxd.ContainerFull) error) error {
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

func (t *ContainerOps) Network(args []string) error {
	return t.run(args, t.printNetwork)
}

func (t *ContainerOps) Wait(args []string) error {
	for _, container := range args {
		var ops Ops
		err := ops.WaitForNetwork(container)
		if err != nil {
			return err
		}
	}
	return nil
}
