package lxdops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"melato.org/lxdops/util"
)

func (t *Config) mergeDescriptions(desc ...string) string {
	var parts []string
	for _, desc := range desc {
		if desc != "" {
			parts = append(parts, desc)
		}
	}
	return strings.Join(parts, "\n")
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

func mergeMaps(a, b map[string]string) map[string]string {
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

func (t *Source) Merge(c *Source) {
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

func (t *ConfigInherit) Merge(c *ConfigInherit) error {
	if t.Project == "" {
		t.Project = c.Project
	}
	if t.Profile == "" {
		t.Profile = c.Profile
	}
	t.Properties = mergeMaps(t.Properties, c.Properties)
	t.ProfileConfig = mergeMaps(t.ProfileConfig, c.ProfileConfig)

	t.Source.Merge(&c.Source)

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
	return nil
}

func (t *Config) merge(file string, included map[string]bool) error {
	if _, found := included[file]; found {
		fmt.Fprintf(os.Stderr, "ignoring duplicate include: %s\n", file)
		return nil
	}
	config, err := ReadRawConfig(file)
	if err != nil {
		return err
	}
	dir := filepath.Dir(file)
	config.ResolvePaths(dir)
	if len(included) == 0 {
		t.ConfigTop = config.ConfigTop

	}
	included[file] = true
	if t.OS == nil {
		t.OS = config.OS
	} else {
		err := t.OS.Merge(config.OS)
		if err != nil {
			return err
		}
	}
	for _, f := range config.Include {
		err := t.merge(string(f), included)
		if err != nil {
			return err
		}
	}
	err = t.ConfigInherit.Merge(&config.ConfigInherit)
	if err != nil {
		return err
	}
	return nil
}

func (t *ConfigInherit) removeDuplicates() {
	// remove duplicate strings
	t.Packages = util.StringSlice(t.Packages).RemoveDuplicates()
	t.Passwords = util.StringSlice(t.Passwords).RemoveDuplicates()
	// how about Require, Devices, Users, Scripts?
}

func ReadConfig(file string) (*Config, error) {
	result := &Config{}
	included := make(map[string]bool)
	err := result.merge(file, included)
	if err != nil {
		return nil, err
	}
	if result.OS == nil {
		result.OS = &OS{}
	}
	return result, err
}
