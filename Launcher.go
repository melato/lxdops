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
	return t.ConfigOptions.Run(t.launchContainer, args...)
}

func (t *Launcher) Rebuild(args []string) error {
	t.Trace = true
	return t.ConfigOptions.Run(t.rebuildContainer, args...)
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
	server, err := t.Client.ProjectServer(config.Project)
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
	if config.Devices != nil {
		if len(profiles) == 0 {
			profiles = append(profiles, "default")
		}
		profiles = append(profiles, profileName)
	}
	script := t.NewScript()
	if config.Origin == "" {
		var lxcArgs []string
		if config.Project != "" {
			lxcArgs = append(lxcArgs, "--project", config.Project)
		}
		lxcArgs = append(lxcArgs, "launch")

		osVersion := config.OS.Version
		if osVersion == "" {
			return errors.New("Missing version")
		}
		lxcArgs = append(lxcArgs, osType.ImageName(osVersion))
		for _, profile := range profiles {
			lxcArgs = append(lxcArgs, "-p", profile)
		}
		for _, option := range config.LxcOptions {
			lxcArgs = append(lxcArgs, option)
		}
		lxcArgs = append(lxcArgs, name)
		script.Run("lxc", lxcArgs...)
		if script.HasError() {
			return script.Error()
		}
	} else {
		sourceConfig, err := config.GetSourceConfig()
		if err != nil {
			return err
		}
		var copyArgs []string
		sourceProject := sourceConfig.Project
		if sourceProject == "" {
			sourceProject = config.Project
		}
		if sourceProject != "" {
			copyArgs = append(copyArgs, "--project", sourceProject)
		}
		copyArgs = append(copyArgs, "copy")

		if config.Project != "" {
			copyArgs = append(copyArgs, "--target-project", config.Project)
		}
		sn := SplitSnapshotName(config.Origin)
		if sn.Snapshot == "" {
			copyArgs = append(copyArgs, "--container-only", sn.Container)
		} else {
			copyArgs = append(copyArgs, sn.Container+"/"+sn.Snapshot)
		}
		copyArgs = append(copyArgs, name)
		script.Run("lxc", copyArgs...)
		if script.HasError() {
			return script.Error()
		}

		c, _, err := server.GetContainer(name)
		if err != nil {
			return AnnotateLXDError(name, err)
		}
		c.Profiles = profiles
		op, err := server.UpdateContainer(name, c.ContainerPut, "")
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return AnnotateLXDError(name, err)
		}

		op, err = server.UpdateContainerState(name, api.ContainerStatePut{Action: "start"}, "")
		if err != nil {
			return AnnotateLXDError(name, err)
		}

		if script.HasError() {
			return script.Error()
		}
		if !t.DryRun {
			err := WaitForNetwork(server, name)
			if err != nil {
				return err
			}
		}
	}
	t.NewConfigurer().ConfigureContainer(config, name)
	if config.Snapshot != "" {
		fmt.Printf("snapshot %s %s\n", name, config.Snapshot)
		if !t.DryRun {
			op, err := server.CreateContainerSnapshot(name, api.ContainerSnapshotsPost{Name: config.Snapshot})
			if err != nil {
				return AnnotateLXDError(name, err)
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(name, err)
			}
		}
	}
	if config.Stop {
		fmt.Printf("stop %s\n", name)
		if !t.DryRun {
			op, err := server.UpdateContainerState(name, api.ContainerStatePut{Action: "stop"}, "")
			if err != nil {
				return AnnotateLXDError(name, err)
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(name, err)
			}
		}
	}
	return script.Error()
}

func (t *Launcher) DeleteContainer(config *Config, name string) error {
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	if !t.DryRun {
		op, err := server.DeleteContainer(name)
		if err == nil {
			if t.Trace {
				fmt.Printf("deleted container %s in project %s\n", name, config.Project)
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(name, err)
			}
		} else {
			state, _, err := server.GetContainerState(name)
			if err == nil {
				return errors.New(fmt.Sprintf("container %s is %s", name, state.Status))
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
		for _, fs := range filesystems {
			fmt.Println(fs.Path)
		}
	}
	return nil
}

func (t *Launcher) Delete(args []string) error {
	t.Trace = true
	return t.ConfigOptions.Run(t.deleteContainer, args...)
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
	server, err := t.Client.ProjectServer(config.Project)
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
	return t.ConfigOptions.Run(t.printFilesystems, arg)
}

func (t *Launcher) printDevices(name string, config *Config) error {
	t.updateConfig(config)
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.PrintDevices(name)
}

func (t *Launcher) PrintDevices(arg string) error {
	return t.ConfigOptions.Run(t.printDevices, arg)
}
