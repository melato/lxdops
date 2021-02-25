package lxdops

import (
	"fmt"

	"melato.org/lxdops/util"
)

type ContainerOps struct {
	Client *LxdClient `name:"-"`
}

func (t *ContainerOps) Profiles(container string) error {
	server, err := t.Client.Server()
	if err != nil {
		return err
	}
	c, _, err := server.GetContainer(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	for _, profile := range c.Profiles {
		fmt.Println(profile)
	}
	return nil
}

func (t *ContainerOps) Network(container string) error {
	server, err := t.Client.Server()
	if err != nil {
		return err
	}
	state, _, err := server.GetContainerState(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	for name, net := range state.Network {
		for _, a := range net.Addresses {
			fmt.Printf("%s %s %s %s/%s\n", name, a.Family, a.Scope, a.Address, a.Netmask)
		}
	}
	return nil
}

func (t *ContainerOps) Wait(args []string) error {
	for _, container := range args {
		err := t.Client.WaitForNetwork(container)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *ContainerOps) State(name string, action ...string) error {
	server, container, err := t.Client.ContainerServer(name)
	if err != nil {
		return err
	}
	state, etag, err := server.GetContainerState(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	fmt.Println(etag)
	util.PrintYaml(state)
	return nil
}
