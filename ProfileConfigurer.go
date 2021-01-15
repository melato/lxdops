package lxdops

import (
	"fmt"
	"strings"

	"melato.org/lxdops/util"
	"melato.org/script"
)

type ProfileConfigurer struct {
	Ops           Ops
	ConfigOptions ConfigOptions
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *ProfileConfigurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *ProfileConfigurer) Configured() error {
	if t.DryRun {
		t.Ops.Trace = true
	}
	return nil
}

func (t *ProfileConfigurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Ops.Trace, DryRun: t.DryRun}
}

func (t *ProfileConfigurer) Profiles(name string, config *Config) []string {
	return append(config.Profiles, config.ProfileName(name))
}

func (t *ProfileConfigurer) diffProfiles(name string, config *Config) error {
	c, err := ListContainer(name)
	if err != nil {
		fmt.Println(err)
		return nil
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

func (t *ProfileConfigurer) reorderProfiles(name string, config *Config) error {
	c, err := ListContainer(name)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	profiles := t.Profiles(name, config)
	if util.StringSlice(profiles).Equals(c.Profiles) {
		return nil
	}

	sortedProfiles := util.StringSlice(profiles).Sorted()
	sortedContainer := util.StringSlice(c.Profiles).Sorted()
	if util.StringSlice(sortedProfiles).Equals(sortedContainer) {
		script := t.NewScript()
		script.Run("lxc", "profile", "apply", name, strings.Join(profiles, ","))
		return script.Error
	}
	fmt.Println("profiles differ: " + name)
	return nil
}

func (t *ProfileConfigurer) applyProfiles(name string, config *Config) error {
	profiles := t.Profiles(name, config)
	script := t.NewScript()
	script.Run("lxc", "profile", "apply", name, strings.Join(profiles, ","))
	return script.Error
}

func (t *ProfileConfigurer) listProfiles(name string, config *Config) error {
	for _, profile := range t.Profiles(name, config) {
		fmt.Println(profile)
	}
	return nil
}

func (t *ProfileConfigurer) Apply(args []string) error {
	return t.ConfigOptions.Run(args, t.applyProfiles)
}

func (t *ProfileConfigurer) List(args []string) error {
	return t.ConfigOptions.Run(args, t.listProfiles)
}

func (t *ProfileConfigurer) Diff(args []string) error {
	return t.ConfigOptions.Run(args, t.diffProfiles)
}

func (t *ProfileConfigurer) Reorder(args []string) error {
	return t.ConfigOptions.Run(args, t.reorderProfiles)
}
