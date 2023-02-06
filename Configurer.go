package lxdops

import (
	"fmt"
	"os"

	"melato.org/cloudconfig"
	"melato.org/cloudconfiglxd"
	"melato.org/lxdops/lxdutil"
)

type Configurer struct {
	Client *lxdutil.LxdClient `name:"-"`
	ConfigOptions
	Trace      bool     `name:"trace,t" usage:"print exec arguments"`
	DryRun     bool     `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Components []string `name:"components" usage:"which components to configure: packages, scripts, users"`
	All        bool     `name:"all" usage:"If true, configure all parts, except those that are mentioned explicitly, otherwise configure only parts that are mentioned"`
	Packages   bool     `name:"packages" usage:"whether to install packages"`
	Scripts    bool     `name:"scripts" usage:"whether to run scripts"`
	Files      bool     `name:"files" usage:"whether to push files"`
	Users      bool     `name:"users" usage:"whether to create users and change passwords"`
}

func (t *Configurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *Configurer) Configured() error {
	return t.ConfigOptions.Configured()
}

func (t *Configurer) includes(flag bool) bool {
	if t.All {
		return !flag
	} else {
		return flag
	}
}

/** run things inside the container:  install packages, create users, run scripts */
func (t *Configurer) ConfigureContainer(instance *Instance) error {
	config := instance.Config
	container := instance.Container()
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	if !t.DryRun {
		err := lxdutil.WaitForNetwork(server, container)
		if err != nil {
			return err
		}
	}
	if len(config.CloudConfigFiles) > 0 {
		base := cloudconfiglxd.NewInstanceConfigurer(server, instance.Name)
		base.Log = os.Stdout
		configurer := cloudconfig.NewConfigurer(base)
		configurer.OS = config.OS.Type()
		if configurer.OS == nil {
			return fmt.Errorf("unsupported OS type: %s", config.OS.Name)
		}
		configurer.Log = os.Stdout
		files := make([]string, len(config.CloudConfigFiles))
		for i, file := range config.CloudConfigFiles {
			files[i] = string(file)
		}
		err := configurer.ApplyConfigFiles(files...)
		if err != nil {
			return err
		}
	}
	return nil
}
