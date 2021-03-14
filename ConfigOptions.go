package lxdops

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"melato.org/lxdops/util"
)

type ConfigOptions struct {
	Project          string   `name:"project" usage:"the LXD project to use.  Overrides Config.Project"`
	Name             string   `name:"name" usage:"The name of the container to launch or configure.  If missing, use a separate container for each config, using the name of the config."`
	Properties       []string `name:"P" usage:"a command-line property in the form <key>=<value>.  Command-line properties override instance and global properties"`
	PropertiesFile   string   `name:"properties" usage:"a file containing global config properties.  Instance properties override global properties"`
	properties       map[string]string
	GlobalProperties map[string]string `name:"-"`
	lxc_config
}

func (t *ConfigOptions) Init() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	t.PropertiesFile = filepath.Join(configDir, "lxdops", "properties.yaml")
	return nil
}

func (t *ConfigOptions) Configured() error {
	if t.PropertiesFile != "" {
		_, err := os.Stat(t.PropertiesFile)
		if err == nil {
			return util.ReadYaml(t.PropertiesFile, &t.GlobalProperties)
		}
	}
	return nil
}

func (t *ConfigOptions) initProperties() error {
	if t.properties != nil {
		return nil
	}
	t.properties = make(map[string]string)
	for _, property := range t.Properties {
		i := strings.Index(property, "=")
		if i < 0 {
			return errors.New("missing value from property: " + property)
		}
		t.properties[property[0:i]] = property[i+1:]
	}
	return nil
}

func (t *ConfigOptions) UpdateConfig(config *Config) {
	if t.Project != "" {
		config.Project = t.Project
	}
	if config.Project == "" {
		config.Project = t.CurrentProject()
	}
	for key, value := range t.properties {
		if config.Properties == nil {
			config.Properties = make(map[string]string)
		}
		config.Properties[key] = value
	}
}

func (t *ConfigOptions) ReadConfig(file string) (*Config, error) {
	err := t.initProperties()
	if err != nil {
		return nil, err
	}
	var config *Config
	config, err = ReadConfig(file)
	if err != nil {
		return nil, err
	}

	if !config.Verify() {
		return nil, errors.New("invalid config")
	}
	t.UpdateConfig(config)
	return config, nil
}

func BaseName(file string) string {
	name := filepath.Base(file)
	ext := filepath.Ext(name)
	return name[0 : len(name)-len(ext)]
}

func (t *ConfigOptions) Instance(file string) (*Instance, error) {
	var name string
	if t.Name != "" {
		name = t.Name
	} else {
		name = BaseName(file)
	}
	config, err := t.ReadConfig(file)
	if err != nil {
		return nil, err
	}
	return NewInstance(t.GlobalProperties, config, name)
}

func (t *ConfigOptions) RunInstances(f func(*Instance) error, args ...string) error {
	if t.Name != "" && len(args) != 1 {
		return errors.New("--name can be used with only one config file")
	}
	for _, arg := range args {
		instance, err := t.Instance(arg)
		if err != nil {
			return err
		}
		err = f(instance)
		if err != nil {
			return errors.New(arg + ": " + err.Error())
		}
	}
	return nil
}

func (t *ConfigOptions) InstanceFunc(f func(*Instance) error, multiple bool) func(configs []string) error {
	return func(configs []string) error {
		return t.RunInstances(f, configs...)
	}
}
