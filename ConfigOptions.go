package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"
)

const (
	DefaultProfileSuffix = "lxdops"
)

type ConfigOptions struct {
	ProfilePattern Pattern `name:"profile-pattern" usage:"pattern for device profiles, overrides config"`
	ProfileSuffix  string  `name:"profile-suffix" usage:"suffix for device profiles, overrides config"`
	Name           string  `name:"name" usage:"The name of the container to launch or configure.  If missing, use a separate container for each config, using the name of the config."`
}

func (t *ConfigOptions) Init() error {
	return nil
}

func (t *ConfigOptions) UpdateConfig(config *Config) {
	pattern := t.ProfilePattern
	if pattern == "" && t.ProfileSuffix != "" {
		pattern = Pattern("(container)." + t.ProfileSuffix)
	}
	if pattern != "" {
		if t.ProfileSuffix != "" && t.ProfilePattern != "" {
			fmt.Println("profile-pattern overrides profile-suffix")
		}
		config.Profile = pattern
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
	return name[0 : len(name)-len(ext)]
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

func (t *ConfigOptions) FuncMultiple(f func(string, *Config) error) func(configs []string) error {
	return func(configs []string) error { return t.Run(f, configs...) }
}
