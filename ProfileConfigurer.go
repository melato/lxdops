package lxdops

import (
	"fmt"
	"strings"

	"melato.org/lxdops/lxdutil"
	"melato.org/lxdops/util"
)

type ProfileConfigurer struct {
	Client *lxdutil.LxdClient
	ConfigOptions
	Config bool `name:"config" usage:"use config profiles"`
	Trace  bool
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *ProfileConfigurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *ProfileConfigurer) Configured() error {
	if t.DryRun {
		t.Trace = true
	}
	return t.ConfigOptions.Configured()
}

func (t *ProfileConfigurer) Profiles(instance *Instance) ([]string, error) {
	profile := instance.ProfileName()
	profiles := instance.Config.Profiles
	if t.Config {
		profiles = instance.Config.GetProfilesConfig(profiles)
	}
	return append(profiles, profile), nil
}

func (t *ProfileConfigurer) Diff(instance *Instance) error {
	container := instance.Container()
	server, err := t.Client.ProjectServer(instance.Config.Project)
	if err != nil {
		return err
	}
	c, _, err := server.GetInstance(container)
	if err != nil {
		return lxdutil.AnnotateLXDError(container, err)
	}
	profiles, err := t.Profiles(instance)
	if err != nil {
		return err
	}
	if util.StringSlice(profiles).Equals(c.Profiles) {
		return nil
	}
	onlyInConfig := util.StringSlice(profiles).Diff(c.Profiles)
	onlyInContainer := util.StringSlice(c.Profiles).Diff(profiles)
	sep := " "
	if len(onlyInConfig) > 0 {
		fmt.Printf("%s profiles only in config: %s\n", container, strings.Join(onlyInConfig, sep))
	}
	if len(onlyInContainer) > 0 {
		fmt.Printf("%s profiles only in container: %s\n", container, strings.Join(onlyInContainer, sep))
	}
	if len(onlyInConfig) == 0 && len(onlyInContainer) == 0 {
		fmt.Printf("%s profiles are in different order: %s\n", container, strings.Join(profiles, sep))
	}
	return nil
}

func (t *ProfileConfigurer) Reorder(instance *Instance) error {
	container := instance.Container()
	server, err := t.Client.ProjectServer(instance.Config.Project)
	if err != nil {
		return err
	}
	c, etag, err := server.GetInstance(container)
	if err != nil {
		return lxdutil.AnnotateLXDError(container, err)
	}
	profiles, err := t.Profiles(instance)
	if err != nil {
		return err
	}
	if util.StringSlice(profiles).Equals(c.Profiles) {
		return nil
	}

	sortedProfiles := util.StringSlice(profiles).Sorted()
	sortedContainer := util.StringSlice(c.Profiles).Sorted()
	if util.StringSlice(sortedProfiles).Equals(sortedContainer) {
		c.Profiles = sortedProfiles
		op, err := server.UpdateInstance(container, c.InstancePut, etag)
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return lxdutil.AnnotateLXDError(container, err)
		}
	}
	fmt.Println("profiles differ: " + container)
	return nil
}

func (t *ProfileConfigurer) Apply(instance *Instance) error {
	container := instance.Container()
	server, err := t.Client.ProjectServer(instance.Config.Project)
	if err != nil {
		return err
	}
	c, etag, err := server.GetInstance(container)
	if err != nil {
		return lxdutil.AnnotateLXDError(container, err)
	}
	c.Profiles, err = t.Profiles(instance)
	if err != nil {
		return err
	}
	op, err := server.UpdateInstance(container, c.InstancePut, etag)
	if err != nil {
		return err
	}
	if err := op.Wait(); err != nil {
		return lxdutil.AnnotateLXDError(container, err)
	}
	return nil
}

func (t *ProfileConfigurer) List(instance *Instance) error {
	profiles, err := t.Profiles(instance)
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		fmt.Println(profile)
	}
	return nil
}
