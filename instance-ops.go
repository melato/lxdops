package lxdops

import (
	"fmt"
	"os"

	"melato.org/export/table3"
)

type InstanceOps struct {
	ConfigOptions
	Trace  bool
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *InstanceOps) Configured() error {
	if t.DryRun {
		t.Trace = true
	}
	return nil
}

func (t *InstanceOps) Verify(instance *Instance) error {
	fmt.Println(instance.Name)
	return nil
}

// Print the description of a config file.
func (t *InstanceOps) Description(instance *Instance) error {
	fmt.Println(instance.Config.Description)
	return nil
}

func (t *InstanceOps) Properties(instance *Instance) error {
	instance.Properties.ShowHelp(os.Stdout)
	return nil
}

func (t *InstanceOps) Filesystems(instance *Instance) error {
	filesystems, err := instance.FilesystemList()
	if err != nil {
		return err
	}
	writer := &table.FixedWriter{Writer: os.Stdout}
	var fs InstanceFS
	writer.Columns(
		table.NewColumn("FILESYSTEM", func() interface{} { return fs.Id }),
		table.NewColumn("PATH", func() interface{} { return fs.Path }),
		table.NewColumn("PATTERN", func() interface{} { return fs.Filesystem.Pattern }),
	)
	for _, fs = range filesystems {
		writer.WriteRow()
	}
	writer.End()
	return nil
}

func (t *InstanceOps) Devices(instance *Instance) error {
	writer := &table.FixedWriter{Writer: os.Stdout}
	devices, err := instance.DeviceList()
	if err != nil {
		return err
	}
	var d InstanceDevice
	writer.Columns(
		table.NewColumn("SOURCE", func() interface{} { return d.Source }),
		table.NewColumn("PATH", func() interface{} { return d.Device.Path }),
		table.NewColumn("NAME", func() interface{} { return d.Name }),
		table.NewColumn("DIR", func() interface{} { return d.Device.Dir }),
		table.NewColumn("FILESYSTEM", func() interface{} { return d.Device.Filesystem }),
	)
	for _, d = range devices {
		writer.WriteRow()
	}
	writer.End()
	return nil
}

func (t *InstanceOps) Project(instance *Instance) error {
	fmt.Println(instance.Config.Project)
	return nil
}
