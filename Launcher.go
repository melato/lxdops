package lxdops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/lxdutil"
	"melato.org/lxdops/util"
	"melato.org/script"
)

type Launcher struct {
	Client *lxdutil.LxdClient `name:"-"`
	ConfigOptions
	RebuildProfiles bool `name:"rebuild-profiles" usage:"if true, rebuild profiles according to config, otherwise keep existing profiles"`
	WaitInterval    int  `name:"wait" usage:"# seconds to wait before snapshot"`
	Trace           bool `name:"t" usage:"trace print what is happening"`
	Api             bool `name:"api" usage:"use LXD API to copy containers"`
	DryRun          bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *Launcher) Init() error {
	t.WaitInterval = 5
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

func (t *Launcher) getRebuildOptions(instance *Instance) (error, *RebuildOptions) {
	config := instance.Config
	container := instance.Container()
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err, nil
	}
	options := &RebuildOptions{}
	if !t.RebuildProfiles {
		c, _, err := server.GetInstance(container)
		if err != nil {
			// assume container doesn't exist.  ignore error, empty options
			return nil, options
		}
		options.Profiles = c.Profiles
	}
	state, _, err := server.GetContainerState(container)
	if err != nil {
		// assume container doesn't exist.  ignore error, empty options
		return nil, options
	}
	for network, networkState := range state.Network {
		if networkState.Hwaddr == "" {
			continue
		}
		if options.Hwaddresses == nil {
			options.Hwaddresses = make(map[string]string)
		}
		options.Hwaddresses[network] = networkState.Hwaddr
	}
	return nil, options
}

func (t *Launcher) Rebuild(instance *Instance) error {
	t.Trace = true
	err, options := t.getRebuildOptions(instance)
	if err != nil {
		return err
	}
	err = t.deleteContainer(instance, true)
	if err != nil {
		return err
	}
	return t.launchContainer(instance, options)
}

func (t *Launcher) NewConfigurer() *Configurer {
	var c = &Configurer{Client: t.Client, Trace: t.Trace, DryRun: t.DryRun, All: true}
	return c
}

func (t *Launcher) lxcLaunch(instance *Instance, server lxd.InstanceServer, options *launch_options) error {
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
	lxcArgs = append(lxcArgs, "init")

	image, err := config.OS.Image.Substitute(instance.Properties)
	if err != nil {
		return err
	}

	if image == "" {
		osVersion, err := config.OS.Version.Substitute(instance.Properties)
		if err != nil {
			return err
		}
		if osVersion != "" {
			image = osType.ImageName(osVersion)
		} else {
			return errors.New("Please provide image or version")
		}
	}
	lxcArgs = append(lxcArgs, image)
	for _, profile := range options.Profiles {
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
	return t.configureContainer(instance, server, options)
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
				return lxdutil.AnnotateLXDError(profile, err)
			}
		}
	}
	return nil
}

type RebuildOptions struct {
	Hwaddresses map[string]string
	Profiles    []string
}

type launch_options struct {
	Profiles []string
	RebuildOptions
}

// configureContainer configures the container directly, if necessary, and starts it
func (t *Launcher) configureContainer(instance *Instance, server lxd.InstanceServer, options *launch_options) error {
	container := instance.Container()
	config := instance.Config
	profileName := instance.ProfileName()
	if !t.DryRun {
		c, etag, err := server.GetInstance(container)
		if err != nil {
			return lxdutil.AnnotateLXDError(container, err)
		}
		c.Profiles = options.Profiles
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
						fmt.Printf("remove source device: %s\n", name)
					}
				}
			}

			for name, device := range devices {
				c.Devices[name] = device
				if t.Trace {
					fmt.Printf("add device: %s\n", name)
				}
			}
			for key, value := range config.ProfileConfig {
				c.Config[key] = value
				if t.Trace {
					fmt.Printf("config %s = %s\n", key, value)
				}
			}
		}
		for network, hwaddr := range options.Hwaddresses {
			key := "volatile." + network + ".hwaddr"
			c.InstancePut.Config[key] = hwaddr
			if t.Trace {
				fmt.Printf("set config %s: %s\n", key, hwaddr)
			}
		}
		op, err := server.UpdateInstance(container, c.InstancePut, etag)
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return lxdutil.AnnotateLXDError(container, err)
		}
	}
	if t.Trace {
		fmt.Printf("start %s\n", container)
	}
	if !t.DryRun {
		err := (lxdutil.InstanceServer{server}).StartContainer(container)
		if err != nil {
			return err
		}

		err = lxdutil.WaitForNetwork(server, container)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Launcher) copyContainer(instance *Instance, source ContainerSource, server lxd.InstanceServer, options *launch_options) error {
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
		return fmt.Errorf("%s_%s: %v", source.Project, source.Container, err)
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

	if t.Api && source.Snapshot != "" {
		entry, _, err := sourceServer.GetInstanceSnapshot(source.Container, source.Snapshot)
		if err != nil {
			return err
		}

		// Prepare the instance creation request
		args := lxd.InstanceSnapshotCopyArgs{
			Name: container,
			Mode: "pull",
		}

		op, err := server.CopyInstanceSnapshot(sourceServer, source.Container, *entry, &args)
		if err != nil {
			return err
		}
		err = op.Wait()
		if err != nil {
			return err
		}
	} else {
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

	}
	err = t.configureContainer(instance, server, options)
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
	dev, err := NewDeviceConfigurer(t.Client, instance)
	if err != nil {
		return err
	}
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	return dev.ConfigureDevices(instance)
}

func (t *Launcher) CreateProfile(instance *Instance) error {
	dev, err := NewDeviceConfigurer(t.Client, instance)
	if err != nil {
		return err
	}
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun
	profileName := instance.ProfileName()
	if profileName != "" {
		fmt.Println(profileName)
		return dev.CreateProfile(instance)
	} else {
		fmt.Printf("skipping instance %s: no lxdops profile\n", instance.Name)
		return nil
	}

}

func (t *Launcher) LaunchContainer(instance *Instance) error {
	return t.launchContainer(instance, nil)
}

func (t *Launcher) launchContainer(instance *Instance, rebuildOptions *RebuildOptions) error {
	fmt.Println("launch", instance.Name)
	t.Trace = true
	config := instance.Config
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	dev, err := NewDeviceConfigurer(t.Client, instance)
	if err != nil {
		return err
	}
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
	if len(rebuildOptions.Profiles) > 0 {
		profiles = make([]string, len(rebuildOptions.Profiles))
		for i, profile := range rebuildOptions.Profiles {
			profiles[i] = profile
		}
	} else {
		profiles = append(profiles, config.Profiles...)
		if config.Devices != nil {
			if len(profiles) == 0 {
				profiles = append(profiles, "default")
			}
			if profileName != "" {
				profiles = append(profiles, profileName)
			}
		}
	}
	options := &launch_options{Profiles: profiles}
	if rebuildOptions != nil {
		options.RebuildOptions = *rebuildOptions
	}
	container := instance.Container()
	source := instance.ContainerSource()
	fmt.Printf("source:%v\n", source)
	if !source.IsDefined() {
		err := t.lxcLaunch(instance, server, options)
		if err != nil {
			return err
		}
	} else {
		err := t.copyContainer(instance, *source, server, options)
		if err != nil {
			return err
		}
	}
	configurer := t.NewConfigurer()
	err = configurer.ConfigureContainer(instance)
	if err != nil {
		return err
	}
	if config.Stop || config.Snapshot != "" {
		if t.WaitInterval != 0 {
			fmt.Printf("waiting %d seconds for container installation scripts to complete\n", t.WaitInterval)
			time.Sleep(time.Duration(t.WaitInterval) * time.Second)
		}
	}
	if config.Stop {
		fmt.Printf("stop %s\n", container)
		if !t.DryRun {
			err = (lxdutil.InstanceServer{server}).StopContainer(container)
			if err != nil {
				return err
			}
		}
	}
	if config.Snapshot != "" {
		fmt.Printf("snapshot %s %s\n", container, config.Snapshot)
		if !t.DryRun {
			op, err := server.CreateContainerSnapshot(container, api.ContainerSnapshotsPost{Name: config.Snapshot})
			if err != nil {
				return lxdutil.AnnotateLXDError(container, err)
			}
			if err := op.Wait(); err != nil {
				return lxdutil.AnnotateLXDError(container, err)
			}
		}
	}
	return configurer.PullFiles(instance)
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
			err = (lxdutil.InstanceServer{server}).StopContainer(container)
		}

		op, err := server.DeleteInstance(container)
		if err == nil {
			if t.Trace {
				fmt.Printf("deleted container %s in project %s\n", container, config.Project)
			}
			if err := op.Wait(); err != nil {
				return lxdutil.AnnotateLXDError(container, err)
			}
		} else {
			state, _, err := server.GetInstanceState(container)
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
		if (lxdutil.InstanceServer{server}).ProfileExists(profileName) {
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
		cmd := exec.Command("zfs", append([]string{"list", "-o", "name,used,referenced,origin,mountpoint"}, zfsFilesystems...)...)
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

func (t *Launcher) DestroyContainer(instance *Instance) error {
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
		if fs.Filesystem.Destroy {
			if fs.IsZfs() {
				zfsFilesystems = append(zfsFilesystems, fs.Path)
			} else {
				dirFilesystems = append(dirFilesystems, fs.Path)
			}
		}
	}
	if len(zfsFilesystems) > 0 {
		s := script.Script{DryRun: t.DryRun, Trace: t.Trace}
		lines := s.Cmd("zfs", append([]string{"list", "-H", "-o", "name"}, zfsFilesystems...)...).ToLines()
		s.Errors.Clear()
		for _, line := range lines {
			s.Run("sudo", "zfs", "destroy", "-r", line)
		}
		if s.HasError() {
			return s.Error()
		}
	}
	var firstError error
	for _, dir := range dirFilesystems {
		err := os.RemoveAll(dir)
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
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
	dev, err := NewDeviceConfigurer(t.Client, instance)
	if err != nil {
		return err
	}
	dev.Trace = t.Trace
	dev.DryRun = t.DryRun

	containerName := instance.Container()
	newContainerName := newInstance.Container()
	var container *api.Container
	if len(instance.Config.Devices) > 0 {
		_, _, err := server.GetProfile(newprofile)
		if err == nil {
			return errors.New(fmt.Sprintf("profile %s already exists", newprofile))
		}
		container, _, err = server.GetContainer(containerName)
		if err != nil {
			return lxdutil.AnnotateLXDError(containerName, err)
		}
	}
	if !t.DryRun {
		op, err := server.RenameContainer(containerName, api.ContainerPost{Name: newInstance.Container()})
		if err != nil {
			return lxdutil.AnnotateLXDError(containerName, err)
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
		}
		if !t.DryRun {
			op, err := server.UpdateContainer(newContainerName, container.ContainerPut, "")
			if err != nil {
				return lxdutil.AnnotateLXDError(newContainerName, err)
			}
			if err := op.Wait(); err != nil {
				return lxdutil.AnnotateLXDError(newContainerName, err)
			}
		}
		if t.Trace {
			fmt.Printf("delete profile %s\n", oldprofile)
		}
		if !t.DryRun {
			err = server.DeleteProfile(oldprofile)
			if err != nil {
				return lxdutil.AnnotateLXDError(oldprofile, err)
			}
		}
	}
	return nil
}
