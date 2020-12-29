package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"melato.org/export/program"
)

type Launcher struct {
	Ops            *Ops     `name:""`
	ProfileSuffix  string   `name:"profile-suffix" usage:"suffix for device profiles, if not specified in config"`
	Origin         string   `name:"origin" usage:"container to copy, overrides config"`
	DeviceTemplate string   `name:"device-template" usage:"device dir or dataset to copy, overrides config"`
	DeviceOrigin   string   `name:"device-origin" usage:"zfs snapshot to clone into target device, overrides config"`
	DryRun         bool     `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Profiles       []string `name:"profile,p" usage:"profiles to add to lxc launch"`
	Multiple       bool     `name:"m" usage:"launch each yaml file as a separate container with derived name"`
	Ext            string   `name:"ext" usage:"extension for config files with -m option"`
	Options        []string `name:"X" usage:"additional options to pass to lxc"`
	prog           program.Params
}

func (t *Launcher) Init() error {
	t.ProfileSuffix = "devices"
	return nil
}

func (t *Launcher) Configured() error {
	t.prog.DryRun = t.DryRun
	t.prog.Trace = t.Ops.Trace
	return nil
}

func (op *Launcher) updateConfig(config *Config) {
	if op.Origin != "" {
		config.Origin = op.Origin
	}
	if op.DeviceTemplate != "" {
		config.DeviceTemplate = op.DeviceTemplate
	}
	if op.DeviceOrigin != "" {
		config.DeviceOrigin = op.DeviceOrigin
	}
	if config.ProfileSuffix == "" {
		config.ProfileSuffix = op.ProfileSuffix
	}
}

func (op *Launcher) LaunchOne(name string, configFiles []string) error {
	var err error
	var config *Config
	config, err = ReadConfigs(configFiles...)
	if err != nil {
		return err
	}
	fmt.Println("suffix", config.ProfileSuffix, op.ProfileSuffix)
	op.updateConfig(config)
	fmt.Println("updated", config.ProfileSuffix, op.ProfileSuffix)
	return op.LaunchContainer(config, name)
}

func BaseName(file string) string {
	name := filepath.Base(file)
	ext := filepath.Ext(name)
	if len(ext) == 0 {
		return file
	}
	return name[0 : len(name)-len(ext)]
}

func (op *Launcher) LaunchMultiple(args []string) error {
	for _, arg := range args {
		var name, file string
		if op.Ext == "" {
			file = arg
			name = BaseName(arg)
		} else {
			name = filepath.Base(arg)
			file = arg + "." + op.Ext
		}
		fmt.Println(name, file)
		var err error
		var config *Config
		config, err = ReadConfigs(file)
		if err != nil {
			return err
		}
		op.updateConfig(config)
		err = op.LaunchContainer(config, name)
		if err != nil {
			fmt.Println("failed:", file)
			return err
		}
	}
	return nil
}

func (op *Launcher) Run(args []string) error {
	if op.Multiple {
		return op.LaunchMultiple(args)
	}
	if len(args) < 2 {
		return errors.New("Usage: {name} {configfile}...")
	}
	return op.LaunchOne(args[0], args[1:])
}

func (t *Launcher) NewConfigurer() *Configurer {
	var c = &Configurer{ops: t.Ops, DryRun: t.DryRun, prog: t.prog, All: true}
	return c
}

func (t *Launcher) LaunchContainer(config *Config, name string) error {
	if !config.Verify() {
		return errors.New("prerequisites not met")
	}
	var err error
	osType := config.OS.Type()
	if osType == nil {
		return errors.New("unsupported OS type: " + config.OS.Name)
	}

	dev := NewDeviceConfigurer(t.Ops)
	dev.SetDryRun(t.DryRun)
	err = dev.ConfigureDevices(config, name)
	if err != nil {
		return err
	}
	for _, rep := range config.Repositories {
		t.Ops.CloneRepository(rep)
	}
	var profiles []string
	profiles = append(profiles, config.Profiles...)
	//profiles := []string{"default", "dev", "opt", "tools"}
	if config.Devices != nil {
		if len(profiles) == 0 {
			profiles = append(profiles, "default")
		}
		profiles = append(profiles, config.ProfileName(name))
	}
	fmt.Println("profiles", profiles)
	containerTemplate := config.Origin
	if containerTemplate == "" {
		var lxcArgs []string
		lxcArgs = append(lxcArgs, "launch")

		osVersion := config.OS.Version
		if osVersion == "" {
			return errors.New("Missing version")
		}
		lxcArgs = append(lxcArgs, osType.ImageName(osVersion))
		for _, profile := range profiles {
			lxcArgs = append(lxcArgs, "-p", profile)
		}
		for _, profile := range t.Profiles {
			lxcArgs = append(lxcArgs, "-p", profile)
		}
		for _, option := range t.Options {
			lxcArgs = append(lxcArgs, option)
		}
		lxcArgs = append(lxcArgs, name)
		err = t.prog.NewProgram("lxc").Run(lxcArgs...)
		if err != nil {
			return err
		}
	} else {
		copyArgs := []string{"copy"}
		if !strings.Contains(containerTemplate, "/") {
			copyArgs = append(copyArgs, "--container-only")
		}
		copyArgs = append(copyArgs, containerTemplate, name)
		err = t.prog.NewProgram("lxc").Run(copyArgs...)
		if err != nil {
			return err
		}
		copyArgs = append(copyArgs, containerTemplate)

		err = t.prog.NewProgram("lxc").Run("profile", "apply", name, strings.Join(profiles, ","))
		if err != nil {
			return err
		}
		err = t.prog.NewProgram("lxc").Run("start", name)
		if err != nil {
			return err
		}
		err = t.Ops.waitForNetwork(name)
		if err != nil {
			return err
		}
	}
	t.NewConfigurer().ConfigureContainer(config, name)
	if config.Snapshot != "" {
		err = t.prog.NewProgram("lxc").Run("snapshot", name, config.Snapshot)
	}
	if config.Stop {
		err = t.prog.NewProgram("lxc").Run("stop", name)
	}
	return err
}
