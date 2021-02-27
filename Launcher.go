package lxdops

import (
	"errors"
	"fmt"
	"os"

	"github.com/lxc/lxd/shared/api"
	"melato.org/script/v2"
)

type Launcher struct {
	Client        *LxdClient `name:"-"`
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
	var c = &Configurer{Client: t.Client, Trace: t.Trace, DryRun: t.DryRun, All: true}
	return c
}

func (t *Launcher) LaunchContainer(config *Config, name string) error {
	if !config.Verify() {
		return errors.New("prerequisites not met")
	}
	osType := config.OS.Type()
	if osType == nil {
		return errors.New("unsupported OS type: " + config.OS.Name)
	}
	server, container, err := t.Client.ContainerServer(name)
	if err != nil {
		return err
	}
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	err = dev.ConfigureDevices(name)
	if err != nil {
		return err
	}
	err = dev.CreateProfile(name)
	if err != nil {
		return err
	}

	profileName := config.ProfileName(name)
	var profiles []string
	profiles = append(profiles, config.Profiles...)
	//profiles := []string{"default", "dev", "opt", "tools"}
	if config.Devices != nil {
		if len(profiles) == 0 {
			profiles = append(profiles, "default")
		}
		profiles = append(profiles, profileName)
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

		c, _, err := server.GetContainer(container)
		if err != nil {
			return AnnotateLXDError(container, err)
		}
		c.Profiles = profiles
		op, err := server.UpdateContainer(container, c.ContainerPut, "")
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return AnnotateLXDError(container, err)
		}

		op, err = server.UpdateContainerState(container, api.ContainerStatePut{Action: "start"}, "")
		if err != nil {
			return AnnotateLXDError(container, err)
		}

		if script.HasError() {
			return script.Error()
		}
		if !t.DryRun {
			err := t.Client.WaitForNetwork(name)
			if err != nil {
				return err
			}
		}
	}
	t.NewConfigurer().ConfigureContainer(config, name)
	if config.Snapshot != "" {
		fmt.Printf("snapshot %s %s\n", container, config.Snapshot)
		if !t.DryRun {
			op, err := server.CreateContainerSnapshot(container, api.ContainerSnapshotsPost{Name: config.Snapshot})
			if err != nil {
				return AnnotateLXDError(container, err)
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(container, err)
			}
		}
	}
	if config.Stop {
		fmt.Printf("stop %s\n", container)
		if !t.DryRun {
			op, err := server.UpdateContainerState(container, api.ContainerStatePut{Action: "stop"}, "")
			if err != nil {
				return AnnotateLXDError(container, err)
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(container, err)
			}
		}
	}
	return script.Error()
}

func (t *Launcher) DeleteContainer(config *Config, name string) error {
	server, container, err := t.Client.ContainerServer(name)
	if err != nil {
		return err
	}
	if !t.DryRun {
		op, err := server.DeleteContainer(container)
		if err == nil {
			if t.Trace {
				fmt.Printf("deleted container %s\n", container)
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(container, err)
			}
		} else {
			state, _, err := server.GetContainerState(container)
			if err == nil {
				return errors.New(fmt.Sprintf("container %s is %s", container, state.Status))
			}
		}
	}
	profileName := config.ProfileName(name)
	if !t.DryRun {
		err := server.DeleteProfile(profileName)
		if err == nil && t.Trace {
			fmt.Printf("deleted profile %s\n", profileName)
		}
	}
	return nil
}

func (t *Launcher) deleteContainer(name string, config *Config) error {
	t.updateConfig(config)
	err := t.DeleteContainer(config, name)
	if err != nil {
		return err
	}
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	filesystems, err := dev.ListFilesystems(name)
	if err != nil {
		return err
	}
	if len(filesystems) > 0 {
		fmt.Fprintln(os.Stderr, "not deleted filesystems:")
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

func (t *Launcher) Rename(configFile string, newname string) error {
	oldname := t.ConfigOptions.Name
	if oldname == "" {
		oldname = BaseName(configFile)
	}
	if t.Trace {
		fmt.Printf("rename container %s -> %s\n", oldname, newname)
	}
	if oldname == newname {
		return errors.New("cannot rename to the same name")
	}
	config, err := t.ConfigOptions.ReadConfig(configFile)
	if err != nil {
		return err
	}
	oldprofile := config.ProfileName(oldname)
	newprofile := config.ProfileName(newname)
	server, err := t.Client.Server()
	if err != nil {
		return err
	}
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun

	var container *api.Container
	if len(config.Devices) > 0 {
		_, _, err := server.GetProfile(newprofile)
		if err == nil {
			return errors.New(fmt.Sprintf("profile %s already exists", newprofile))
		}
		container, _, err = server.GetContainer(oldname)
		if err != nil {
			return AnnotateLXDError(oldname, err)
		}
	}
	if !t.DryRun {
		op, err := server.RenameContainer(oldname, api.ContainerPost{Name: newname})
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return err
		}
	}
	if len(config.Devices) > 0 {
		err = dev.RenameFilesystems(oldname, newname)
		if err != nil {
			return err
		}
		err = dev.CreateProfile(newname)
		if err != nil {
			return err
		}
		var replaced bool
		for i, profile := range container.Profiles {
			if profile == oldprofile {
				container.Profiles[i] = newprofile
				replaced = true
				break
			}
		}
		if !replaced {
			container.Profiles = append(container.Profiles, newprofile)
		}
		if t.Trace {
			fmt.Printf("apply %s profiles: %v\n", newname, container.Profiles)
			fmt.Printf("delete profile %s\n", oldprofile)
		}
		if !t.DryRun {
			op, err := server.UpdateContainer(newname, container.ContainerPut, "")
			if err != nil {
				return err
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(newname, err)
			}
			err = server.DeleteProfile(oldprofile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Launcher) printFilesystems(name string, config *Config) error {
	t.updateConfig(config)
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.PrintFilesystems(name)
}

func (t *Launcher) PrintFilesystems(arg string) error {
	return t.ConfigOptions.Run([]string{arg}, t.printFilesystems)
}
