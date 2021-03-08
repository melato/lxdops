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
	filesystems, err := instance.Filesystems()
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
	var deviceName string
	var d *Device
	writer.Columns(
		table.NewColumn("SOURCE", func() interface{} {
			dir, err := instance.DeviceDir(deviceName, d)
			if err != nil {
				return err
			}
			return dir
		}),
		table.NewColumn("PATH", func() interface{} { return d.Path }),
		table.NewColumn("NAME", func() interface{} { return deviceName }),
		table.NewColumn("DIR", func() interface{} { return d.Dir }),
		table.NewColumn("FILESYSTEM", func() interface{} { return d.Filesystem }),
	)
	for deviceName, d = range instance.Config.Devices {
		writer.WriteRow()
	}
	writer.End()
	return nil
}

func (t *InstanceOps) Project(instance *Instance) error {
	fmt.Println(instance.Config.Project)
	return nil
}
