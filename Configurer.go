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
	Trace  bool `name:"trace,t" usage:"print exec arguments"`
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *Configurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *Configurer) Configured() error {
	return t.ConfigOptions.Configured()
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
