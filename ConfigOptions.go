package lxdops

import (
	"errors"
	"path/filepath"
	"strings"
)

const (
	DefaultProfileSuffix = "lxdops"
)

type ConfigOptions struct {
	Project    string   `name:"project" usage:"the LXD project to use.  Overrides Config.Project"`
	Name       string   `name:"name" usage:"The name of the container to launch or configure.  If missing, use a separate container for each config, using the name of the config."`
	Properties []string `name:"D" usage:"Define a config property in the form <key>=<value>"`
	properties map[string]string
	lxc_config
}

func (t *ConfigOptions) Init() error {
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

func (t *ConfigOptions) RunInstances(f func(*Instance) error, args ...string) error {
	return t.Run(func(name string, config *Config) error {
		return f(config.NewInstance(name))
	}, args...)
}

func (t *ConfigOptions) Run(f func(name string, config *Config) error, args ...string) error {
	if t.Name != "" && len(args) != 1 {
		return errors.New("--name can be used with only one config file")
	}
	for _, arg := range args {
		var name, file string
		file = arg
		if t.Name != "" && len(args) == 1 {
			name = t.Name
		} else {
			name = BaseName(arg)
		}
		config, err := t.ReadConfig(file)
		if err != nil {
			return err
		}
		err = f(name, config)
		if err != nil {
			return errors.New(file + ": " + err.Error())
		}
	}
	return nil
}

func (t *ConfigOptions) Func(f func(string, *Config) error) func(config string) error {
	return func(config string) error { return t.Run(f, config) }
}

func (t *ConfigOptions) InstanceFunc(f func(*Instance) error, multiple bool) func(configs []string) error {
	return func(configs []string) error {
		return t.RunInstances(f, configs...)
	}
}
