package lxdops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

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
	return t.ConfigOptions.Configured()
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

func (t *Launcher) lxcLaunch(instance *Instance, server lxd.InstanceServer, profiles []string) error {
	config := instance.Config
	osType := config.OS.Type()
	if osType == nil {
		return errors.New("unsupported OS type: " + config.OS.Name)
	}
	s := t.NewScript()
	container := instance.Container()
	profileName := instance.ProfileName()
	var lxcArgs []string
	if config.Project != "" {
		lxcArgs = append(lxcArgs, "--project", config.Project)
	}
	if profileName != "" {
		lxcArgs = append(lxcArgs, "launch")
	} else {
		lxcArgs = append(lxcArgs, "init")
	}

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
	if s.HasError() {
		return s.Error()
	}
	if profileName == "" {
		return t.configureContainer(instance, server, profiles)
	}
	return nil
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

// configureContainer configures the container directly, if necessary, and starts it
func (t *Launcher) configureContainer(instance *Instance, server lxd.InstanceServer, profiles []string) error {
	container := instance.Container()
	config := instance.Config
	profileName := instance.ProfileName()
	if !t.DryRun {
		c, _, err := server.GetContainer(container)
		if err != nil {
			return AnnotateLXDError(container, err)
		}
		c.Profiles = profiles
		if profileName == "" {
			// there is no lxdops profile.  configure container directly
			devices, err := instance.NewDeviceMap()
			if err != nil {
				return err
			}
			source := instance.DeviceSource()
			if source.IsDefined() {
				// remove old devices, specified by source config
				for name, _ := range source.Instance.Config.Devices {
					delete(c.Devices, name)
					if t.Trace {
						fmt.Println("remove source device: %s\n", name)
					}
				}
			}

			for name, device := range devices {
				c.Devices[name] = device
				if t.Trace {
					fmt.Println("add device: %s\n", name)
				}
			}
			for key, value := range config.ProfileConfig {
				c.Config[key] = value
				if t.Trace {
					fmt.Println("config %s = %s\n", key, value)
				}
			}
		}
		op, err := server.UpdateContainer(container, c.ContainerPut, "")
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return AnnotateLXDError(container, err)
		}
	}
	if t.Trace {
		fmt.Printf("start %s\n", container)
	}
	if !t.DryRun {
		err := (InstanceServer{server}).StartContainer(container)
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

func (t *Launcher) copyContainer(instance *Instance, source ContainerSource, server lxd.InstanceServer, profiles []string) error {
	s := t.NewScript()
	container := instance.Container()
	config := instance.Config
	sourceServer, err := t.Client.ProjectServer(source.Project)
	if err != nil {
		return err
	}
	allProfiles, err := server.GetProfileNames()
	if err != nil {
		return err
	}
	c, _, err := sourceServer.GetContainer(source.Container)
	if err != nil {
		return AnnotateLXDError(source.Container, err)
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
	if source.Project != "" {
		copyArgs = append(copyArgs, "--project", source.Project)
	}

	copyArgs = append(copyArgs, "copy")

	if config.Project != "" {
		copyArgs = append(copyArgs, "--target-project", config.Project)
	}
	if source.Snapshot == "" {
		copyArgs = append(copyArgs, "--instance-only", source.Container)
	} else {
		copyArgs = append(copyArgs, source.Container+"/"+source.Snapshot)
	}
	copyArgs = append(copyArgs, container)
	s.Run("lxc", copyArgs...)
	if s.HasError() {
		t.deleteProfiles(server, missingProfiles)
		return s.Error()
	}

	err = t.configureContainer(instance, server, profiles)
	err2 := t.deleteProfiles(server, missingProfiles)
	if err != nil {
		return err
	}
	if err2 != nil {
		return err
	}
	return nil
}

func (t *Launcher) CreateDevices(instance *Instance) error {
	t.Trace = true
	dev := NewDeviceConfigurer(t.Client, instance.Config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.ConfigureDevices(instance)
}

func (t *Launcher) CreateProfile(instance *Instance) error {
	dev := NewDeviceConfigurer(t.Client, instance.Config)
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	profileName := instance.ProfileName()
	if profileName != "" {
		fmt.Println(profileName)
		return dev.CreateProfile(instance)
	} else {
		fmt.Println("skipping instance %s: no lxdops profile\n", instance.Name)
		return nil
	}

}

func (t *Launcher) LaunchContainer(instance *Instance) error {
	fmt.Println("launch", instance.Name)
	t.Trace = true
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

	profileName := instance.ProfileName()
	if profileName != "" {
		err = dev.CreateProfile(instance)
		if err != nil {
			return err
		}
	}

	var profiles []string
	profiles = append(profiles, config.Profiles...)
	if config.Devices != nil {
		if len(profiles) == 0 {
			profiles = append(profiles, "default")
		}
		if profileName != "" {
			profiles = append(profiles, profileName)
		}
	}
	container := instance.Container()
	source := instance.ContainerSource()
	fmt.Printf("source:%v\n", source)
	if !source.IsDefined() {
		err := t.lxcLaunch(instance, server, profiles)
		if err != nil {
			return err
		}
	} else {
		err := t.copyContainer(instance, *source, server, profiles)
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
			err = (InstanceServer{server}).StopContainer(container)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
			err = (InstanceServer{server}).StopContainer(container)
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
	profileName := instance.ProfileName()

	if !t.DryRun {
		err := server.DeleteProfile(profileName)
		if err == nil && t.Trace {
			fmt.Printf("delete profile %s\n", profileName)
		}
	} else {
		if (InstanceServer{server}).ProfileExists(profileName) {
			fmt.Printf("delete profile %s\n", profileName)
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
	var zfsFilesystems []string
	var dirFilesystems []string
	for _, fs := range filesystems {
		if fs.IsZfs() {
			zfsFilesystems = append(zfsFilesystems, fs.Path)
		} else {
			dirFilesystems = append(dirFilesystems, fs.Path)
		}
	}
	fmt.Fprintln(os.Stderr, "remaining filesystems:")
	if len(zfsFilesystems) > 0 {
		cmd := exec.Command("zfs", append([]string{"list", "-o", "name,used,referenced,mountpoint"}, zfsFilesystems...)...)
		cmd.Stderr = io.Discard
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	if len(dirFilesystems) > 0 {
		cmd := exec.Command("ls", append([]string{"-l"}, dirFilesystems...)...)
		cmd.Stderr = io.Discard
		cmd.Stdout = os.Stdout
		cmd.Run()
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
	oldprofile := instance.ProfileName()
	newInstance, err := instance.NewInstance(newname)
	if err != nil {
		return err
	}
	newprofile := newInstance.ProfileName()
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
