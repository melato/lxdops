package lxdops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"melato.org/lxdops/util"
)

type ConfigReader struct {
	included map[string]bool
}

func (r *ConfigReader) isIncluded(file string) bool {
	return r.included[file]
}

func (r *ConfigReader) addIncluded(file string) {
	if r.included == nil {
		r.included = make(map[string]bool)
	}
	r.included[file] = true
}

func (t *OS) Merge(c *OS) error {
	if c == nil {
		// keep the one we have
		return nil
	}
	if t.Name != c.Name {
		return errors.New("cannot merge incompatible OSs: " + t.Name + ", " + c.Name)
	} else if t.Version != c.Version {
		if t.Version == "" {
			t.Version = c.Version
		} else if c.Version == "" {
			// keep the one we have
		} else {
			return errors.New("cannot merge incompatible os versions: " + t.Version + ", " + c.Version)
		}
	}
	return nil
}

func (r *ConfigReader) mergeMaps(a, b map[string]string) map[string]string {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		a = make(map[string]string)
	}
	for key, value := range b {
		a[key] = value
	}
	return a
}

func (r *ConfigReader) mergeSource(t, c *Source) {
	if t.Origin == "" {
		t.Origin = c.Origin
	}
	if t.DeviceTemplate == "" {
		t.DeviceTemplate = c.DeviceTemplate
	}
	if t.DeviceOrigin == "" {
		t.DeviceOrigin = c.DeviceOrigin
	}
	if t.SourceConfig == "" {
		t.SourceConfig = c.SourceConfig
	}
}

func (r *ConfigReader) mergeInherit(t, c *ConfigInherit) {
	if t.Project == "" {
		t.Project = c.Project
	}
	if t.Container == "" {
		t.Container = c.Container
	}
	if t.Profile == "" {
		t.Profile = c.Profile
	}
	t.Properties = r.mergeMaps(t.Properties, c.Properties)
	t.ProfileConfig = r.mergeMaps(t.ProfileConfig, c.ProfileConfig)

	r.mergeSource(&t.Source, &c.Source)

	if len(t.LxcOptions) == 0 {
		t.LxcOptions = c.LxcOptions
	}

	if t.Filesystems == nil {
		t.Filesystems = make(map[string]*Filesystem)
	}
	for id, fs := range c.Filesystems {
		t.Filesystems[id] = fs
	}
	if t.Devices == nil {
		t.Devices = make(map[string]*Device)
	}
	for id, d := range c.Devices {
		t.Devices[id] = d
	}
	t.PreScripts = append(t.PreScripts, c.PreScripts...)
	t.Packages = append(t.Packages, c.Packages...)
	t.Profiles = append(t.Profiles, c.Profiles...)
	t.Users = append(t.Users, c.Users...)
	t.Files = append(t.Files, c.Files...)
	t.Scripts = append(t.Scripts, c.Scripts...)
	t.Passwords = append(t.Passwords, c.Passwords...)

	t.removeDuplicates()
}

func (r *ConfigReader) mergeFile(t *Config, file string) error {
	if r.isIncluded(file) {
		fmt.Fprintf(os.Stderr, "ignoring duplicate include: %s\n", file)
		return nil
	}
	config, err := ReadRawConfig(file)
	if err != nil {
		return err
	}
	dir := filepath.Dir(file)
	config.ResolvePaths(dir)
	if len(r.included) == 0 {
		t.ConfigTop = config.ConfigTop

	}
	r.addIncluded(file)
	if t.OS == nil {
		t.OS = config.OS
	} else {
		err := t.OS.Merge(config.OS)
		if err != nil {
			return err
		}
	}
	for _, f := range config.Include {
		err := r.mergeFile(config, string(f))
		if err != nil {
			return err
		}
	}
	r.mergeInherit(&t.ConfigInherit, &config.ConfigInherit)
	return nil
}

func (t *ConfigInherit) removeDuplicates() {
	// remove duplicate strings
	t.Packages = util.StringSlice(t.Packages).RemoveDuplicates()
	t.Passwords = util.StringSlice(t.Passwords).RemoveDuplicates()
	// how about Require, Devices, Users, Scripts?
}

func (r *ConfigReader) Read(file string) (*Config, error) {
	result := &Config{}
	err := r.mergeFile(result, file)
	if err != nil {
		return nil, err
	}
	if result.OS == nil {
		result.OS = &OS{}
	}
	return result, err
}

func ReadConfig(file string) (*Config, error) {
	r := &ConfigReader{}
	return r.Read(file)
}
