package lxdops

import (
	"errors"
	"fmt"
	"strings"

	"melato.org/export/program"
)

type Launcher struct {
	Ops            *Ops     `name:""`
	ProfileDir     string   `name:"profile-dir" usage:"directory to save profile files"`
	Origin         string   `name:"origin" usage:"container to copy, overrides config"`
	DeviceTemplate string   `name:"device-template" usage:"device dir or dataset to copy, overrides config"`
	DeviceOrigin   string   `name:"device-origin" usage:"zfs snapshot to clone into target device, overrides config"`
	DryRun         bool     `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Profiles       []string `name:"profile,p" usage:"profiles to add to lxc launch"`
	Options        []string `name:"X" usage:"additional options to pass to lxc"`
	prog           program.Params
}

func (t *Launcher) Init() error {
	t.ProfileDir = "target"
	return nil
}

func (t *Launcher) Configured() error {
	t.prog.DryRun = t.DryRun
	t.prog.Trace = t.Ops.Trace
	return nil
}

func (op *Launcher) Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: {name} {configfile}...")
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
	if op.Origin != "" {
		config.Origin = op.Origin
	}
	if op.DeviceTemplate != "" {
		config.DeviceTemplate = op.DeviceTemplate
	}
	if op.DeviceOrigin != "" {
		config.DeviceOrigin = op.DeviceOrigin
	}
	return op.LaunchContainer(config, name)
}

func (t *Launcher) NewConfigurer() *Configurer {
	var c = &Configurer{ops: t.Ops, DryRun: t.DryRun, prog: t.prog, All: true}
	return c
}

func (t *Launcher) LaunchContainer(config *Config, name string) error {
	var err error
	osType := config.OS.Type()
	if osType == nil {
		return errors.New("unsupported OS type: " + config.OS.Name)
	}

	dev := &DeviceConfigurer{Ops: t.Ops, ProfileDir: t.ProfileDir, DryRun: t.DryRun}
	dev.Configured()
	err = dev.CreateDeviceDirs(config, name)
	if err != nil {
		return err
	}
	zfsroot, err := t.Ops.ZFSRoot()
	if err != nil {
		return err
	}
	err = config.CreateProfile(name, t.ProfileDir, zfsroot)
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
