package lxdops

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type Launcher struct {
	ConfigOptions ConfigOptions
	Trace         bool `name:""` //`name:"trace,t" usage:"print exec arguments"`
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

func (t *Launcher) rebuildContainer(name string, config *Config) error {
	t.updateConfig(config)
	exec.Command("lxc", "stop", name).Run() // ignore error
	err := t.DeleteContainer(config, name)
	if err != nil {
		return err
	}
	return t.LaunchContainer(config, name)
}

func (t *Launcher) Launch(args []string) error {
	t.Trace = true
	return t.ConfigOptions.Run(args, t.launchContainer)
}

func (t *Launcher) Rebuild(args []string) error {
	t.Trace = true
	return t.ConfigOptions.Run(args, t.rebuildContainer)
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

	dev := NewDeviceConfigurer(config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	err = dev.ConfigureDevices(name)
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
		if script.HasError() {
			return script.Error()
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
		copyArgs = append(copyArgs, container)
		script.Run("lxc", copyArgs...)
		if script.HasError() {
			return script.Error()
		}

		script.Run("lxc", append(projectArgs, "profile", "apply", container, strings.Join(profiles, ","))...)
		script.Run("lxc", append(projectArgs, "start", container)...)
		if script.HasError() {
			return script.Error()
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
	return script.Error()
}

func (t *Launcher) DeleteContainer(config *Config, name string) error {
	project, container := SplitContainerName(name)
	s := t.NewScript()
	s.Errors.AlwaysContinue = true
	projectArgs := ProjectArgs(project)
	s.Run("lxc", append(projectArgs, "delete", container)...)
	s.Run("lxc", "profile", "delete", config.ProfileName(name))
	return s.Error()
}

func (t *Launcher) deleteContainer(name string, config *Config) error {
	t.updateConfig(config)
	err := t.DeleteContainer(config, name)
	if err != nil {
		return err
	}
	dev := NewDeviceConfigurer(config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	filesystems, err := dev.ListFilesystems(name)
	if err != nil {
		return err
	}
	if len(filesystems) > 0 {
		fmt.Println("not deleted filesystems:")
		for _, dir := range filesystems {
			fmt.Println(dir)
		}
	}
	return nil
}

func (t *Launcher) Delete(args []string) error {
	t.Trace = true
	return t.ConfigOptions.Run(args, t.deleteContainer)
}

func (t *Launcher) Rename(oldpath, newpath string) error {
	t.Trace = true
	oldname := BaseName(oldpath)
	newname := BaseName(newpath)
	if oldname == newname {
		return errors.New(fmt.Sprintf("%s and %s have the same name", oldname, newname))
	}
	if filepath.Ext(oldpath) != filepath.Ext(newpath) {
		return errors.New(fmt.Sprintf("%s and %s don't have the same extension", oldpath, newpath))
	}
	if util.FileExists(newpath) {
		return errors.New(fmt.Sprintf("%s already exists", newpath))
	}
	config, err := t.ConfigOptions.ReadConfig(oldpath)
	if err != nil {
		return err
	}
	s := t.NewScript()
	oldprofile := config.ProfileName(oldname)
	newprofile := config.ProfileName(newname)
	if len(config.Devices) > 0 {
		if ProfileExists(newprofile) {
			return errors.New(fmt.Sprintf("profile %s already exists", newprofile))
		}
	}
	s.Run("lxc", "rename", oldname, newname)
	if len(config.Devices) > 0 {
		s.Run("lxc", "profile", "remove", newname, oldprofile)
		s.Run("lxc", "profile", "delete", oldprofile)
	}
	if s.HasError() {
		return s.Error()
	}
	dev := NewDeviceConfigurer(config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	err = dev.RenameFilesystems(oldname, newname)
	if err != nil {
		return err
	}
	if len(config.Devices) > 0 {
		err := dev.CreateProfile(newname)
		if err != nil {
			return err
		}
		s.Run("lxc", "profile", "add", newname, newprofile)
	}
	s.Run("mv", oldpath, newpath)
	return s.Error()
}

func (t *Launcher) printFilesystems(name string, config *Config) error {
	t.updateConfig(config)
	dev := NewDeviceConfigurer(config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.PrintFilesystems(name)
}

func (t *Launcher) PrintFilesystems(arg string) error {
	return t.ConfigOptions.Run([]string{arg}, t.printFilesystems)
}
