package lxdops

import (
	"fmt"
	"os"
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

func (t *ConfigOps) Verify(name string, config *Config) error {
	fmt.Println(name)
	return nil
}

func (t *ConfigOps) CreateDevices(name string, config *Config) error {
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.ConfigureDevices(name)
}

// Print the description of a config file.
func (t *ConfigOps) Description(name string, config *Config) error {
	fmt.Println(config.Description)
	return nil
}

func (t *ConfigOps) Properties(name string, config *Config) error {
	properties := config.NewProperties(name)
	properties.ShowHelp(os.Stdout)
	return nil
}
