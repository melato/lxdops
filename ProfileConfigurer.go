package lxdops

import (
	"fmt"
	"strings"

	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type ProfileConfigurer struct {
	Client        *LxdClient
	ConfigOptions ConfigOptions
	Trace         bool
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *ProfileConfigurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *ProfileConfigurer) Configured() error {
	if t.DryRun {
		t.Trace = true
	}
	return nil
}

func (t *ProfileConfigurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace, DryRun: t.DryRun}
}

func (t *ProfileConfigurer) Profiles(name string, config *Config) []string {
	return append(config.Profiles, config.ProfileName(name))
}

func (t *ProfileConfigurer) diffProfiles(name string, config *Config) error {
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	c, _, err := server.GetContainer(name)
	if err != nil {
		return AnnotateLXDError(name, err)
	}
	profiles := t.Profiles(name, config)
	if util.StringSlice(profiles).Equals(c.Profiles) {
		return nil
	}
	onlyInConfig := util.StringSlice(profiles).Diff(c.Profiles)
	onlyInContainer := util.StringSlice(c.Profiles).Diff(profiles)
	sep := " "
	if len(onlyInConfig) > 0 {
		fmt.Printf("%s profiles only in config: %s\n", name, strings.Join(onlyInConfig, sep))
	}
	if len(onlyInContainer) > 0 {
		fmt.Printf("%s profiles only in container: %s\n", name, strings.Join(onlyInContainer, sep))
	}
	if len(onlyInConfig) == 0 && len(onlyInContainer) == 0 {
		fmt.Printf("%s profiles are in different order: %s\n", name, strings.Join(profiles, sep))
	}
	return nil
}

func (t *ProfileConfigurer) reorderProfiles(container string, config *Config) error {
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	c, _, err := server.GetContainer(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	profiles := t.Profiles(container, config)
	if util.StringSlice(profiles).Equals(c.Profiles) {
		return nil
	}

	sortedProfiles := util.StringSlice(profiles).Sorted()
	sortedContainer := util.StringSlice(c.Profiles).Sorted()
	if util.StringSlice(sortedProfiles).Equals(sortedContainer) {
		c.Profiles = sortedProfiles
		op, err := server.UpdateContainer(container, c.ContainerPut, "")
		if err != nil {
			return err
		}
		if err := op.Wait(); err != nil {
			return AnnotateLXDError(container, err)
		}
	}
	fmt.Println("profiles differ: " + container)
	return nil
}

func (t *ProfileConfigurer) applyProfiles(container string, config *Config) error {
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	c, _, err := server.GetContainer(container)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	c.Profiles = t.Profiles(container, config)
	op, err := server.UpdateContainer(container, c.ContainerPut, "")
	if err != nil {
		return err
	}
	if err := op.Wait(); err != nil {
		return AnnotateLXDError(container, err)
	}
	return nil
}

func (t *ProfileConfigurer) listProfiles(name string, config *Config) error {
	for _, profile := range t.Profiles(name, config) {
		fmt.Println(profile)
	}
	return nil
}

func (t *ProfileConfigurer) Apply(args []string) error {
	return t.ConfigOptions.Run(t.applyProfiles, args...)
}

func (t *ProfileConfigurer) List(arg string) error {
	return t.ConfigOptions.Run(t.listProfiles, arg)
}

func (t *ProfileConfigurer) Diff(args []string) error {
	return t.ConfigOptions.Run(t.diffProfiles, args...)
}

func (t *ProfileConfigurer) Reorder(args []string) error {
	return t.ConfigOptions.Run(t.reorderProfiles, args...)
}
