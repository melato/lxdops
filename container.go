package lxdops

import (
	"fmt"
	"os"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"melato.org/export/table3"
	"melato.org/lxdops/util"
)

type ContainerOps struct {
	Client *LxdClient `name:"-"`
	server lxd.InstanceServer
}

func (t *ContainerOps) Configured() error {
	project := t.Client.CurrentProject()
	server, err := t.Client.ProjectServer(project)
	if err != nil {
		return err
	}
	t.server = server
	return nil
}

func (t *ContainerOps) Profiles(container string) error {
	c, _, err := t.server.GetContainer(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	for _, profile := range c.Profiles {
		fmt.Println(profile)
	}
	return nil
}

func (t *ContainerOps) Network(container string) error {
	server := t.server
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
		err := WaitForNetwork(t.server, container)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *ContainerOps) State(container string, action ...string) error {
	server := t.server
	state, etag, err := server.GetContainerState(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	fmt.Println(etag)
	util.PrintYaml(state)
	return nil
}

func (t *ContainerOps) Devices(container string) error {
	c, _, err := t.server.GetContainer(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	//util.PrintYaml(c.ExpandedDevices)
	writer := &table.FixedWriter{Writer: os.Stdout}
	var deviceName string
	var d map[string]string
	writer.Columns(
		table.NewColumn("PATH", func() interface{} { return d["path"] }),
		table.NewColumn("SOURCE", func() interface{} { return d["source"] }),
		table.NewColumn("NAME", func() interface{} { return deviceName }),
	)
	for deviceName, d = range c.ExpandedDevices {
		if d["type"] == "disk" {
			writer.WriteRow()
		}
	}
	writer.End()
	return nil
}
