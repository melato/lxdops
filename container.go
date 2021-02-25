package lxdops

import (
	"errors"
	"fmt"
	"time"

	"github.com/lxc/lxd/shared/api"
)

type ContainerOps struct {
	Client *LxdClient `name:"-"`
}

func (t *ContainerOps) listProfiles(c *api.ContainerFull) error {
	for _, profile := range c.Profiles {
		fmt.Println(profile)
	}
	return nil
}

func (t *ContainerOps) printNetwork(c *api.ContainerFull) error {
	if c.State != nil {
		for name, net := range c.State.Network {
			for _, a := range net.Addresses {
				fmt.Printf("%s %s %s %s/%s\n", name, a.Family, a.Scope, a.Address, a.Netmask)
			}
		}
	}
	return nil
}

func (t *ContainerOps) run(args []string, f func(c *api.ContainerFull) error) error {
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

func (t *ContainerOps) WaitForNetwork(container string) error {
	server, err := t.Client.Server()
	if err != nil {
		return err
	}
	for i := 0; i < 30; i++ {
		state, _, err := server.GetContainerState(container)
		if err != nil {
			return err
		}
		if state == nil {
			continue
		}
		for _, net := range state.Network {
			for _, a := range net.Addresses {
				if a.Family == "inet" && a.Scope == "global" {
					fmt.Println(a.Address)
					return nil
				}
			}
		}
		fmt.Printf("status: %s\n", state.Status)
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not get ip address for: " + container)
}

func (t *ContainerOps) Wait(args []string) error {
	for _, container := range args {
		err := t.WaitForNetwork(container)
		if err != nil {
			return err
		}
	}
	return nil
}
