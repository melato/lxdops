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

type addressColumns struct {
	net string
	x   *api.ContainerStateNetworkAddress
}

func (t *addressColumns) network() interface{} {
	return t.net
}

func (t *addressColumns) family() interface{} {
	return t.x.Family
}

func (t *addressColumns) scope() interface{} {
	return t.x.Scope
}

func (t *addressColumns) address() interface{} {
	return t.x.Address
}

func (t *addressColumns) netmask() interface{} {
	return t.x.Netmask
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
	var col addressColumns
	writer := &table.FixedWriter{Writer: os.Stdout}
	writer.Columns(
		table.NewColumn("network", col.network),
		table.NewColumn("family", col.family),
		table.NewColumn("scope", col.scope),
		table.NewColumn("address", col.address),
		table.NewColumn("netmask", col.netmask),
	)
	for name, net := range state.Network {
		col.net = name
		for _, a := range net.Addresses {
			col.x = &a
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
