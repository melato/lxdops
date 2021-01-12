package lxdops

import (
	"errors"
)

type DeviceCmd struct {
	Ops           *Ops   `name:""`
	DryRun        bool   `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	ProfileSuffix string `name:"profile-suffix" usage:"suffix for device profiles, if not specified in config"`
}

func (t *DeviceCmd) Init() error {
	t.ProfileSuffix = "devices"
	return nil
}

func (t *DeviceCmd) Configured() error {
	if t.DryRun {
		t.Ops.Trace = true
	}
	return nil
}

func (t *DeviceCmd) Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: {profile-name} {configfile}...")
	}
	name := args[0]
	var err error
	var config *Config
	config, err = ReadConfigs(args[1:]...)
	if err != nil {
		return err
	}
	if !config.Verify() {
		return errors.New("prerequisites not met")
	}
	if config.ProfileSuffix == "" {
		config.ProfileSuffix = t.ProfileSuffix
	}
	dev := NewDeviceConfigurer(t.Ops)
	dev.SetDryRun(t.DryRun)
	return dev.ConfigureDevices(config, name)
}
