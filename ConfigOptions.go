package lxdops

import (
	"errors"
	"path/filepath"
)

const (
	DefaultProfileSuffix = "devices"
)

type ConfigOptions struct {
	ProfileSuffix string `name:"profile-suffix" usage:"suffix for device profiles, if not specified in config"`
	Multiple      bool   `name:"m" usage:"treat each yaml file as a separate container with derived name"`
	Ext           string `name:"ext" usage:"extension for config files with -m option"`
}

func (t *ConfigOptions) Init() error {
	t.ProfileSuffix = DefaultProfileSuffix
	return nil
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
	if config.ProfileSuffix == "" {
		config.ProfileSuffix = t.ProfileSuffix
	}
	return config, nil
}

func (t *ConfigOptions) runMultiple(args []string, f func(name string, config *Config) error) error {
	for _, arg := range args {
		var name, file string
		if t.Ext == "" {
			file = arg
			name = BaseName(arg)
		} else {
			name = filepath.Base(arg)
			file = arg + "." + t.Ext
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

func (t *ConfigOptions) Run(args []string, f func(name string, config *Config) error) error {
	if t.Multiple {
		return t.runMultiple(args, f)
	}
	if len(args) < 2 {
		return errors.New("Usage: {name} {configfile}...")
	}
	name := args[0]
	config, err := t.ReadConfig(args[1:]...)
	if err != nil {
		return err
	}
	return f(name, config)
}
