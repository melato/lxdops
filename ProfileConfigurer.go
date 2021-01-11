package lxdops

import (
	"fmt"
	"strings"

	"melato.org/export/program"
)

type ProfileConfigurer struct {
	ops           *Ops
	ConfigOptions ConfigOptions
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	prog          program.Params
}

func NewProfileConfigurer(ops *Ops) *ProfileConfigurer {
	var t ProfileConfigurer
	t.ops = ops
	return &t
}

func (t *ProfileConfigurer) Init() error {
	return t.ConfigOptions.Init()
	return nil
}

func (t *ProfileConfigurer) Configured() error {
	t.prog.DryRun = t.DryRun
	t.prog.Trace = t.ops.Trace
	return nil
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
	return t.prog.NewProgram("lxc").Run("profile", "apply", name, strings.Join(profiles, ","))
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
