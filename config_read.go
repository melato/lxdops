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

func (t *Config) Merge(c *Config) error {
	//fmt.Printf("Merge %p %v %p %v\n", t, t.OS, c, c.OS)
	if t.OS == nil {
		t.OS = c.OS
	} else if c.OS == nil {
		// keep the one we have
	} else if t.OS.Name != c.OS.Name {
		return errors.New("cannot merge incompatible OSs: " + t.OS.Name + ", " + c.OS.Name)
	} else if t.OS.Version != c.OS.Version {
		if t.OS.Version == "" {
			t.OS.Version = c.OS.Version
		} else if c.OS.Version == "" {
			// keep the one we have
		} else {
			return errors.New("cannot merge incompatible os versions: " + t.OS.Version + ", " + c.OS.Version)
		}
	}
	if t.Project == "" {
		t.Project = c.Project
	}
	if t.Profile == "" {
		t.Profile = c.Profile
	}
	t.Description = t.mergeDescriptions(t.Description, c.Description)
	if t.Origin == "" {
		t.Origin = c.Origin
	}
	if t.DeviceTemplate == "" {
		t.DeviceTemplate = c.DeviceTemplate
	}
	if t.DeviceOrigin == "" {
		t.DeviceOrigin = c.DeviceOrigin
	}
	if t.Properties == nil {
		t.Properties = make(map[string]string)
	}
	for key, value := range c.Properties {
		_, exists := t.Properties[key]
		if !exists {
			t.Properties[key] = value
		}
	}
	t.SourceFilesystems = make(map[string]Pattern)
	if t.SourceFilesystems == nil {
		t.SourceFilesystems = make(map[string]Pattern)
	}
	for key, value := range c.SourceFilesystems {
		t.SourceFilesystems[key] = value
	}
	if t.Snapshot == "" {
		t.Snapshot = c.Snapshot
	}
	if t.SourceConfig == "" {
		t.SourceConfig = c.SourceConfig
	}
	t.Stop = t.Stop || c.Stop
	t.RequiredFiles = append(t.RequiredFiles, c.RequiredFiles...)
	t.Filesystems = append(t.Filesystems, c.Filesystems...)
	t.Devices = append(t.Devices, c.Devices...)
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
	included[file] = true
	config, err := ReadRawConfig(file)
	if err != nil {
		return err
	}
	dir := filepath.Dir(file)
	config.ResolvePaths(dir)
	for _, f := range config.Include {
		err := t.merge(string(f), included)
		if err != nil {
			return err
		}
	}
	err = t.Merge(config)
	if err != nil {
		return err
	}
	return nil
}

func (t *Config) removeDuplicates() {
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

func ReadConfigs(files ...string) (*Config, error) {
	result := &Config{}
	included := make(map[string]bool)
	for _, file := range files {
		err := result.merge(file, included)
		if err != nil {
			return nil, err
		}
	}
	if result.OS == nil {
		result.OS = &OS{}
	}
	return result, nil
}

func ReadConfigs1(files ...string) (*Config, error) {
	// This does not work, and I couldn't figure out why
	// Inside Merge() the lists are merged properly.
	// When Merge() returns result has a different value than the selector inside Merge()
	// and the result of Merge is lost
	if len(files) == 0 {
		return &Config{}, nil
	}
	var result *Config
	for i, file := range files {
		//fmt.Println(file)
		c, err := ReadConfig(file)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			result = c
		} else {
			//fmt.Printf("before merge %p %p %d\n", result, c, len(result.Devices))
			result.Merge(c)
			//fmt.Printf("after merge %p %p %d\n", result, c, len(result.Devices))
		}
	}
	return result, nil
}
