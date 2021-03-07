package lxdops

import (
	"errors"
	"fmt"
	"os"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/util"
	"melato.org/script"
)

type Launcher struct {
	Client *LxdClient `name:"-"`
	ConfigOptions
	Trace  bool `name:"t" usage:"trace print what is happening"`
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
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

func (t *Launcher) Rebuild(instance *Instance) error {
	t.Trace = true
	err := t.deleteContainer(instance, true)
	if err != nil {
		return err
	}
	return t.LaunchContainer(instance)
}

func (t *Launcher) NewConfigurer() *Configurer {
	var c = &Configurer{Client: t.Client, Trace: t.Trace, DryRun: t.DryRun, All: true}
	return c
}

func (t *Launcher) lxcLaunch(instance *Instance, profiles []string) error {
	config := instance.Config
	osType := config.OS.Type()
	if osType == nil {
		return errors.New("unsupported OS type: " + config.OS.Name)
	}
	s := t.NewScript()
	container := instance.Container()
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
	lxcArgs = append(lxcArgs, container)
	s.Run("lxc", lxcArgs...)
	return s.Error()
}

func (t *Launcher) createEmptyProfile(server lxd.InstanceServer, profile string) error {
	post := api.ProfilesPost{Name: profile, ProfilePut: api.ProfilePut{Description: "lxdops placeholder profile"}}
	if t.Trace {
		fmt.Printf("create empty profile %s:\n", profile)
	}
	if !t.DryRun {
		return server.CreateProfile(post)
	}
	return nil
}

func (t *Launcher) deleteProfiles(server lxd.InstanceServer, profiles []string) error {
	// delete the missing profiles from the new container, and delete them
	for _, profile := range profiles {
		if t.Trace {
			fmt.Printf("delete profile %s\n", profile)
		}
		if !t.DryRun {
			err := server.DeleteProfile(profile)
			if err != nil {
				return AnnotateLXDError(profile, err)
			}
		}
	}
	return nil
}

// lxcCopy copy by invoking lxc
func (t *Launcher) lxcCopy(instance *Instance, sourceProject, sourceContainer, snapshot string) error {
	config := instance.Config
	container := instance.Container()
	var copyArgs []string
	if sourceProject != "" {
		copyArgs = append(copyArgs, "--project", sourceProject)
	}

	copyArgs = append(copyArgs, "copy")

	if config.Project != "" {
		copyArgs = append(copyArgs, "--target-project", config.Project)
	}
	if snapshot == "" {
		copyArgs = append(copyArgs, "--container-only", sourceContainer)
	} else {
		copyArgs = append(copyArgs, sourceContainer+"/"+snapshot)
	}
	copyArgs = append(copyArgs, container)
	s := t.NewScript()
	s.Run("lxc", copyArgs...)
	return s.Error()
}

func (t *Launcher) copyContainer(instance *Instance, server lxd.InstanceServer, profiles []string) error {
	s := t.NewScript()
	container := instance.Container()
	config := instance.Config
	sourceConfig, err := config.GetSourceConfig()
	if err != nil {
		return err
	}
	sourceProject := sourceConfig.Project
	if sourceProject == "" {
		sourceProject = config.Project
	}

	sourceServer, err := t.Client.ProjectServer(sourceProject)
	if err != nil {
		return err
	}
	allProfiles, err := server.GetProfileNames()
	if err != nil {
		return err
	}
	sourceContainer, snapshot := SplitSnapshotName(config.Origin)
	if sourceContainer == "" {
		sourceContainer, err = instance.SourceContainer()
		if config.SourceConfig == "" {
			return errors.New("missing origin container")
		}
	}
	c, _, err := sourceServer.GetContainer(sourceContainer)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	missingProfiles := util.StringSlice(c.Profiles).Diff(allProfiles)
	// lxc copy will fail if the source container has profiles that do not exist in the target server
	// so create the missing profiles, and delete them after the copy
	for _, profile := range missingProfiles {
		err := t.createEmptyProfile(server, profile)
		if err != nil {
			return err
		}
	}

	var copyArgs []string
	if sourceProject != "" {
		copyArgs = append(copyArgs, "--project", sourceProject)
	}

	copyArgs = append(copyArgs, "copy")

	if config.Project != "" {
		copyArgs = append(copyArgs, "--target-project", config.Project)
	}
	if snapshot == "" {
		copyArgs = append(copyArgs, "--container-only", sourceContainer)
	} else {
		copyArgs = append(copyArgs, sourceContainer+"/"+snapshot)
	}
	copyArgs = append(copyArgs, container)
	s.Run("lxc", copyArgs...)
	if s.HasError() {
		t.deleteProfiles(server, missingProfiles)
		return s.Error()
	}

	if !t.DryRun {
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
	}
	err = t.deleteProfiles(server, missingProfiles)
	if err != nil {
		return err
	}
	if t.Trace {
		fmt.Printf("start %s\n", container)
	}
	if !t.DryRun {
		err := UpdateContainerState(server, container, "start")
		if err != nil {
			return err
		}

		err = WaitForNetwork(server, container)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Launcher) LaunchContainer(instance *Instance) error {
	config := instance.Config
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	err = dev.ConfigureDevices(instance)
	if err != nil {
		return err
	}
	err = dev.CreateProfile(instance)
	if err != nil {
		return err
	}

	profileName, err := instance.ProfileName()
	if err != nil {
		return err
	}
	var profiles []string
	profiles = append(profiles, config.Profiles...)
	if config.Devices != nil {
		if len(profiles) == 0 {
			profiles = append(profiles, "default")
		}
		profiles = append(profiles, profileName)
	}
	s := t.NewScript()
	container := instance.Container()
	if config.Origin == "" {
		err := t.lxcLaunch(instance, profiles)
		if err != nil {
			return err
		}
	} else {
		err := t.copyContainer(instance, server, profiles)
		if err != nil {
			return err
		}
	}
	t.NewConfigurer().ConfigureContainer(instance)
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
	return s.Error()
}

func (t *Launcher) deleteContainer(instance *Instance, stop bool) error {
	config := instance.Config
	container := instance.Container()
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	if !t.DryRun {
		if stop {
			_ = UpdateContainerState(server, container, "stop")
		}

		op, err := server.DeleteContainer(container)
		if err == nil {
			if t.Trace {
				fmt.Printf("deleted container %s in project %s\n", container, config.Project)
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
	profileName, err := instance.ProfileName()
	if err != nil {
		return err
	}

	if !t.DryRun {
		err := server.DeleteProfile(profileName)
		if err == nil && t.Trace {
			fmt.Printf("deleted profile %s\n", profileName)
		}
	}
	return nil
}

func (t *Launcher) DeleteContainer(instance *Instance) error {
	err := t.deleteContainer(instance, false)
	if err != nil {
		return err
	}
	filesystems, err := instance.FilesystemList()
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

func (t *Launcher) Rename(configFile string, newname string) error {
	instance, err := t.ConfigOptions.Instance(configFile)
	if err != nil {
		return err
	}

	if t.Trace {
		fmt.Printf("rename container %s -> %s\n", instance.Name, newname)
	}
	if instance.Name == newname {
		return errors.New("cannot rename to the same name")
	}
	oldprofile, err := instance.ProfileName()
	if err != nil {
		return err
	}
	newInstance := instance.Config.NewInstance(newname)
	newprofile, err := newInstance.ProfileName()
	if err != nil {
		return err
	}
	server, err := t.Client.ProjectServer(instance.Config.Project)
	if err != nil {
		return err
	}
	dev := NewDeviceConfigurer(t.Client, instance.Config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun

	containerName := instance.Container()
	newContainerName := instance.Container()
	var container *api.Container
	if len(instance.Config.Devices) > 0 {
		_, _, err := server.GetProfile(newprofile)
		if err == nil {
			return errors.New(fmt.Sprintf("profile %s already exists", newprofile))
		}
		container, _, err = server.GetContainer(containerName)
		if err != nil {
			return AnnotateLXDError(containerName, err)
		}
	}
	if !t.DryRun {
		op, err := server.RenameContainer(containerName, api.ContainerPost{Name: newInstance.Container()})
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return err
		}
	}
	if len(instance.Config.Devices) > 0 {
		err = dev.RenameFilesystems(instance, newInstance)
		if err != nil {
			return err
		}
		err = dev.CreateProfile(newInstance)
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
			op, err := server.UpdateContainer(newContainerName, container.ContainerPut, "")
			if err != nil {
				return err
			}
			if err := op.Wait(); err != nil {
				return AnnotateLXDError(newContainerName, err)
			}
			err = server.DeleteProfile(oldprofile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Launcher) CreateDevices(name string, config *Config) error {
	dev := NewDeviceConfigurer(t.Client, config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.ConfigureDevices(config.NewInstance(name))
}
