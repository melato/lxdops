package lxdops

import (
	"fmt"
	"os"

	"github.com/lxc/lxd/shared/api"
	"melato.org/export/table3"
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
	writer := &table.FixedWriter{Writer: os.Stdout}
	var network string
	var a *api.ContainerStateNetworkAddress
	writer.Columns(
		table.NewColumn("NETWORK", func() interface{} { return network }),
		table.NewColumn("FAMILY", func() interface{} { return a.Family }),
		table.NewColumn("SCOPE", func() interface{} { return a.Scope }),
		table.NewColumn("ADDRESS", func() interface{} { return a.Address }),
		table.NewColumn("NETMASK", func() interface{} { return a.Netmask }),
	)
	for name, net := range state.Network {
		network = name
		for _, address := range net.Addresses {
			a = &address
			writer.WriteRow()
		}
	}
	writer.End()
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
