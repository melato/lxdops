package lxdops

import (
	"errors"
	"fmt"
	"strings"

	"melato.org/script"
)

type Launcher struct {
	ConfigOptions ConfigOptions
	Trace         bool `name:"trace,t" usage:"print exec arguments"`
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`

	Origin         string   `name:"origin" usage:"container to copy, overrides config"`
	DeviceTemplate string   `name:"device-template" usage:"device dir or dataset to copy, overrides config"`
	DeviceOrigin   string   `name:"device-origin" usage:"zfs snapshot to clone into target device, overrides config"`
	Profiles       []string `name:"profile,p" usage:"profiles to add to lxc launch"`
	Options        []string `name:"X" usage:"additional options to pass to lxc"`
}

func (t *Launcher) Init() error {
	return t.ConfigOptions.Init()
}

func (t *Launcher) Configured() error {
	if t.DryRun {
		t.Trace = true
	}
	return nil
}

func (t *Launcher) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace, DryRun: t.DryRun}
}

func (t *Launcher) updateConfig(config *Config) {
	if t.Origin != "" {
		config.Origin = t.Origin
	}
	if t.DeviceTemplate != "" {
		config.DeviceTemplate = t.DeviceTemplate
	}
	if t.DeviceOrigin != "" {
		config.DeviceOrigin = t.DeviceOrigin
	}
	t.ConfigOptions.UpdateConfig(config)
}

func (t *Launcher) launchContainer(name string, config *Config) error {
	t.updateConfig(config)
	return t.LaunchContainer(config, name)
}

func (t *Launcher) Launch(args []string) error {
	return t.ConfigOptions.Run(args, t.launchContainer)
}

func (t *Launcher) NewConfigurer() *Configurer {
	var c = &Configurer{Trace: t.Trace, DryRun: t.DryRun, All: true}
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

	dev := &DeviceConfigurer{Trace: t.Trace, DryRun: t.DryRun}
	err = dev.ConfigureDevices(config, name)
	if err != nil {
		return err
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
	containerTemplate := config.Origin
	script := t.NewScript()
	project, container := SplitContainerName(name)
	projectArgs := ProjectArgs(project)
	if containerTemplate == "" {
		lxcArgs := append(projectArgs, "launch")

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
		lxcArgs = append(lxcArgs, container)
		script.Run("lxc", lxcArgs...)
		if script.Error != nil {
			return err
		}
	} else {
		sn := SplitSnapshotName(containerTemplate)
		copyArgs := append(ProjectArgs(sn.Project), "copy")
		if project != "" {
			copyArgs = append(copyArgs, "--target-project", project)
		}
		if sn.Snapshot == "" {
			copyArgs = append(copyArgs, "--container-only", sn.Container)
		} else {
			copyArgs = append(copyArgs, sn.Container+"/"+sn.Snapshot)
		}
		script.Run("lxc", copyArgs...)
		if script.Error != nil {
			return err
		}

		script.Run("lxc", append(projectArgs, "profile", "apply", container, strings.Join(profiles, ","))...)
		script.Run("lxc", append(projectArgs, "start", container)...)
		if script.Error != nil {
			return err
		}
		if !t.DryRun {
			err = WaitForNetwork(name)
			if err != nil {
				return err
			}
		}
	}
	t.NewConfigurer().ConfigureContainer(config, name)
	if config.Snapshot != "" {
		script.Run("lxc", append(projectArgs, "snapshot", container, config.Snapshot)...)
	}
	if config.Stop {
		script.Run("lxc", append(projectArgs, "stop", container)...)
	}
	return script.Error
}

func (t *Launcher) deleteContainer(name string, config *Config) error {
	t.updateConfig(config)
	project, container := SplitContainerName(name)
	var firstError script.Error
	script := t.NewScript()
	projectArgs := ProjectArgs(project)
	script.Run("lxc", append(projectArgs, "delete", container)...)
	firstError.Add(script.Error)
	script.Error = nil
	script.Run("lxc", "profile", "delete", config.ProfileName(name))
	firstError.Add(script.Error)
	script.Error = nil
	dev := &DeviceConfigurer{Trace: t.Trace, DryRun: t.DryRun}
	filesystems, err := dev.ListFilesystems(config, name)
	firstError.Add(err)
	if err == nil && len(filesystems) > 0 {
		fmt.Println("not deleted filesystems:")
		for _, dir := range filesystems {
			fmt.Println(dir)
		}
	}
	return firstError.First
}

func (t *Launcher) Delete(args []string) error {
	return t.ConfigOptions.Run(args, t.deleteContainer)
}
