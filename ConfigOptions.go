package lxdops

import (
	"errors"
	"path/filepath"
)

const (
	DefaultProfileSuffix = "lxdops"
)

type ConfigOptions struct {
	ProfileSuffix string `name:"profile-suffix" usage:"suffix for device profiles, overrides config"`
	Name          string `name:"name" usage:"The name of the container to launch or configure.  If missing, use a separate container for each config, using the name of the config."`
}

func (t *ConfigOptions) Init() error {
	return nil
}

func (t *ConfigOptions) UpdateConfig(config *Config) {
	if config.ProfileSuffix == "" {
		suffix := t.ProfileSuffix
		if suffix == "" {
			suffix = DefaultProfileSuffix
		}
		config.ProfileSuffix = suffix
	}
}

func (t *ConfigOptions) ReadConfig(args ...string) (*Config, error) {
	var err error
	var config *Config
	config, err = ReadConfigs(args...)
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
	if len(ext) == 0 {
		return file
	}
	return name[0 : len(name)-len(ext)]
}

func (t *ConfigOptions) runMultiple(args []string, f func(name string, config *Config) error) error {
	for _, arg := range args {
		var name, file string
		file = arg
		name = BaseName(arg)
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

func (t *ConfigOptions) Run(args []string, f func(name string, config *Config) error) error {
	if t.Name == "" {
		return t.runMultiple(args, f)
	}
	if len(args) < 1 {
		return errors.New("Usage: {config-file}...")
	}
	config, err := t.ReadConfig(args...)
	if err != nil {
		return err
	}
	return f(t.Name, config)
}
