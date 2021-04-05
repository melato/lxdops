package lxdops

import (
	"fmt"
	"os"
	"sort"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/util"
	"melato.org/table3"
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

func (t *ContainerOps) Config(container string) error {
	c, _, err := t.server.GetContainer(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	writer := &table.FixedWriter{Writer: os.Stdout}
	var keys []string
	for key, _ := range c.Config {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var key, value string
	writer.Columns(
		table.NewColumn("KEY", func() interface{} { return key }),
		table.NewColumn("VALUE", func() interface{} { return value }),
	)

	for _, key = range keys {
		value = c.Config[key]
		writer.WriteRow()
	}
	writer.End()
	return nil
}

func (t *ContainerOps) Network(container string) error {
	server := t.server
	state, _, err := server.GetContainerState(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	writer := &table.FixedWriter{Writer: os.Stdout}
	var name string
	var net api.ContainerStateNetwork
	var a *api.ContainerStateNetworkAddress
	writer.Columns(
		table.NewColumn("NETWORK", func() interface{} { return name }),
		table.NewColumn("HWADDR", func() interface{} { return net.Hwaddr }),
		table.NewColumn("FAMILY", func() interface{} { return a.Family }),
		table.NewColumn("SCOPE", func() interface{} { return a.Scope }),
		table.NewColumn("ADDRESS", func() interface{} { return a.Address }),
		table.NewColumn("NETMASK", func() interface{} { return a.Netmask }),
	)
	for name, net = range state.Network {
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

type disk_device struct {
	Name, Source, Path string
	Readonly           string
}
type disk_device_sorter []disk_device

func (t disk_device_sorter) Len() int           { return len(t) }
func (t disk_device_sorter) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t disk_device_sorter) Less(i, j int) bool { return t[i].Source < t[j].Source }

type ContainerDevicesOp struct {
	ContainerOps *ContainerOps `name:""`
	Yaml         bool
}

func (t *ContainerDevicesOp) Devices(container string) error {
	c, _, err := t.ContainerOps.server.GetContainer(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	if t.Yaml {
		util.PrintYaml(c.ExpandedDevices)
		return nil
	}
	writer := &table.FixedWriter{Writer: os.Stdout}

	var devices []disk_device
	for name, d := range c.ExpandedDevices {
		if d["type"] == "disk" {
			devices = append(devices, disk_device{Name: name, Path: d["path"], Source: d["source"], Readonly: d["readonly"]})
		}
	}
	sort.Sort(disk_device_sorter(devices))

	var d disk_device
	writer.Columns(
		table.NewColumn("SOURCE", func() interface{} { return d.Source }),
		table.NewColumn("PATH", func() interface{} { return d.Path }),
		table.NewColumn("NAME", func() interface{} { return d.Name }),
		table.NewColumn("READONLY", func() interface{} { return d.Readonly }),
	)
	for _, d = range devices {
		writer.WriteRow()
	}
	writer.End()
	return nil
}
