package lxdops

import (
	"fmt"
)

type ConfigOps struct {
	Client        *LxdClient `name:"-"`
	ConfigOptions ConfigOptions
	Trace         bool
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *ConfigOps) Configured() error {
	if t.DryRun {
		t.Trace = true
	}
	return nil
}

func (t *ConfigOps) verify(name string, config *Config) error {
	fmt.Println(name)
	return nil
}

func (t *ConfigOps) Verify(args []string) error {
	return t.ConfigOptions.Run(args, t.verify)
}

func (t *ConfigOps) createDevices(name string, config *Config) error {
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.ConfigureDevices(name)
}

func (t *ConfigOps) CreateDevices(args []string) error {
	return t.ConfigOptions.Run(args, t.createDevices)
}

func (t *ConfigOps) printDescription(name string, config *Config) error {
	fmt.Println(name, config.Description)
	return nil
}

/** Print the description of a config file. */
func (t *ConfigOps) Description(args []string) error {
	return t.ConfigOptions.Run(args, t.printDescription)
}
