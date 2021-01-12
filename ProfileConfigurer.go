package lxdops

import (
	"fmt"
	"strings"

	"melato.org/script"
)

type ProfileConfigurer struct {
	ops           *Ops
	ConfigOptions ConfigOptions
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func NewProfileConfigurer(ops *Ops) *ProfileConfigurer {
	var t ProfileConfigurer
	t.ops = ops
	return &t
}

func (t *ProfileConfigurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *ProfileConfigurer) NewScript() *script.Script {
	return &script.Script{Trace: t.ops.Trace, DryRun: t.DryRun}
}

func (t *ProfileConfigurer) Profiles(name string, config *Config) []string {
	return append(config.Profiles, config.ProfileName(name))
}

func (t *ProfileConfigurer) diffProfiles(name string, config *Config) error {
	c, err := ListContainer(name)
	if err != nil {
		return err
	}
	profiles := t.Profiles(name, config)
	if !EqualArrays(profiles, c.Profiles) {
		fmt.Printf("%s profiles: %s\n", name, strings.Join(c.Profiles, ","))
		fmt.Printf("%s config:   %s\n", name, strings.Join(profiles, ","))
	}
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
