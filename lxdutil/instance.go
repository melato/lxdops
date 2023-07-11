package lxdutil

import (
	"fmt"
	"os"
	"sort"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"melato.org/lxdops/yaml"
	table "melato.org/table3"
)

// InstanceOps - operations on LXD instances (formerly ContainerOps)
type InstanceOps struct {
	Client *LxdClient `name:"-"`
	server lxd.InstanceServer
}

func (t *InstanceOps) Configured() error {
	project := t.Client.CurrentProject()
	server, err := t.Client.ProjectServer(project)
	if err != nil {
		return err
	}
	t.server = server
	return nil
}

func (t *InstanceOps) Profiles(instance string) error {
	c, _, err := t.server.GetInstance(instance)
	if err != nil {
		return AnnotateLXDError(instance, err)
	}
	for _, profile := range c.Profiles {
		fmt.Println(profile)
	}
	return nil
}

func (t *InstanceOps) Config(instance string) error {
	c, _, err := t.server.GetInstance(instance)
	if err != nil {
		return AnnotateLXDError(instance, err)
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

func (t *InstanceOps) Network(instance string) error {
	server := t.server
	state, _, err := server.GetInstanceState(instance)
	if err != nil {
		return AnnotateLXDError(instance, err)
	}
	writer := &table.FixedWriter{Writer: os.Stdout}
	var name string
	var net api.InstanceStateNetwork
	var a *api.InstanceStateNetworkAddress
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

func (t *InstanceOps) Wait(args []string) error {
	for _, instance := range args {
		err := WaitForNetwork(t.server, instance)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *InstanceOps) State(instance string) error {
	server := t.server
	state, etag, err := server.GetInstanceState(instance)
	if err != nil {
		return AnnotateLXDError(instance, err)
	}
	fmt.Println(etag)
	yaml.Print(state)
	return nil
}

type Statistics struct {
	Containers int
	Devices    int
}

func (t *InstanceOps) Statistics() error {
	server := t.server
	containers, err := server.GetContainers()
	if err != nil {
		return err
	}
	var st Statistics
	for _, c := range containers {
		st.Containers++
		for _, d := range c.ExpandedDevices {
			if d["type"] == "disk" {
				st.Devices++
			}
		}
	}
	fmt.Printf("Containers: %d\n", st.Containers)
	fmt.Printf("Devices:    %d\n", st.Devices)
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

type InstanceDevicesOp struct {
	InstanceOps *InstanceOps `name:""`
	Yaml        bool
}

func (t *InstanceDevicesOp) Devices(instance string) error {
	c, _, err := t.InstanceOps.server.GetInstance(instance)
	if err != nil {
		return AnnotateLXDError(instance, err)
	}
	if t.Yaml {
		yaml.Print(c.ExpandedDevices)
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

func (t *InstanceOps) Info(instance string) error {
	c, _, err := t.server.GetInstance(instance)
	if err != nil {
		return AnnotateLXDError(instance, err)
	}
	return yaml.Print(c)
}

func (t *InstanceOps) ListHwaddr() error {
	server := t.server
	instances, err := server.GetInstances(api.InstanceTypeAny)
	if err != nil {
		return err
	}
	var i api.Instance
	writer := &table.FixedWriter{Writer: os.Stdout}
	writer.Columns(
		table.NewColumn("HWADDR", func() interface{} { return i.Config["volatile.eth0.hwaddr"] }),
		table.NewColumn("NAME", func() interface{} { return i.Name }),
	)
	for _, i = range instances {
		writer.WriteRow()
	}
	writer.End()
	return nil
}

func (t *InstanceOps) ListImages() error {
	server := t.server
	images, err := server.GetImages()
	if err != nil {
		return err
	}
	fingerprints := make(map[string]string)
	for _, image := range images {
		names := make([]string, len(image.Aliases))
		for i, a := range image.Aliases {
			names[i] = a.Name
		}
		fingerprints[image.Fingerprint] = strings.Join(names, ",")
	}
	instances, err := server.GetInstances(api.InstanceTypeAny)
	if err != nil {
		return err
	}
	var i api.Instance
	writer := &table.FixedWriter{Writer: os.Stdout}
	writer.Columns(
		table.NewColumn("IMAGE", func() interface{} {
			fg := i.Config["volatile.base_image"]
			image := fingerprints[fg]
			if image == "" {
				image = fg
			}
			return image
		}),
		table.NewColumn("NAME", func() interface{} { return i.Name }),
	)
	for _, i = range instances {
		writer.WriteRow()
	}
	writer.End()
	return nil
}
