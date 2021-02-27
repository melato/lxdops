package lxdops

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"

	"melato.org/lxdops/util"
)

func (t *OS) String() string {
	if t.Version == "" {
		return t.Name
	} else {
		return t.Name + "/" + t.Version
	}
}

func (t *OS) Equals(x *OS) bool {
	return t.Name == x.Name && t.Version == x.Version
}

/** Check that the requirements are met */
func (t *Config) VerifyFileExists(file HostPath) bool {
	if file != "" && !util.FileExists(string(file)) {
		fmt.Fprintf(os.Stderr, "file does not exist: %s\n", string(file))
		return false
	}
	return true
}

func (u *User) EffectiveUser() *User {
	if u.Name == "" {
		currentUser, err := user.Current()
		if err == nil {
			var u2 User
			u2 = *u
			u2.Name = currentUser.Username
			if u.Uid == "" {
				u2.Uid = currentUser.Uid
			}
			return &u2
		}
	}
	return u
}

func (user *User) HomeDir() string {
	if user.Name == "root" {
		return "/root"
	}
	if user.Home != "" {
		return user.Home
	} else {
		return "/home/" + user.Name
	}
}

func (config *Config) verifyDevices() bool {
	valid := true
	filesystems := make(map[string]bool)
	for _, fs := range config.Filesystems {
		if filesystems[fs.Id] {
			valid = false
			fmt.Fprintf(os.Stderr, "duplicate filesystem: %s\n", fs.Id)
		}
		filesystems[fs.Id] = true
	}

	deviceNames := make(map[string]bool)
	devicePaths := make(map[string]bool)
	for _, d := range config.Devices {
		if !filesystems[d.Filesystem] {
			valid = false
			fmt.Fprintf(os.Stderr, "unknown filesystem id: %s\n", d.Filesystem)
		}
		if deviceNames[d.Name] {
			valid = false
			fmt.Fprintf(os.Stderr, "duplicate device name: %s\n", d.Name)
		}
		deviceNames[d.Name] = true
		if devicePaths[d.Path] {
			valid = false
			fmt.Fprintf(os.Stderr, "duplicate device path: %s\n", d.Path)
		}
		deviceNames[d.Path] = true
	}
	return valid
}

func (config *Config) Verify() bool {
	valid := true
	for _, u := range config.Users {
		if !u.IsValidName() {
			valid = false
			fmt.Fprintf(os.Stderr, "invalid user name: %s\n", u.Name)
		}
	}
	for _, file := range config.RequiredFiles {
		if !config.VerifyFileExists(file) {
			valid = false
		}
	}
	for _, file := range config.Files {
		if !config.VerifyFileExists(file.Source) {
			valid = false
		}
	}
	if !config.VerifyFileExists(config.SourceConfig) {
		valid = false
	}
	if !config.verifyDevices() {
		valid = false
	}
	return valid
}

func (u *User) IsValidName() bool {
	re := regexp.MustCompile("^[A-Za-z][A-Za-z0-9_]+$")
	return u.Name == "" || re.MatchString(u.Name)
}

func (t *Config) Print() error {
	return PrintConfigYaml(t)
}

/** Read config without includes */
func ReadRawConfig(file string) (*Config, error) {
	return ReadConfigYaml(file)
}

func (t *Config) Merge(c *Config) error {
	//fmt.Printf("Merge %p %v %p %v\n", t, t.OS, c, c.OS)
	if t.OS == nil {
		t.OS = c.OS
	} else if c.OS == nil {
		// keep the one we have
	} else if t.OS.Name != c.OS.Name {
		return errors.New("cannot merge incompatible oses: " + t.OS.Name + ", " + c.OS.Name)
	} else if t.OS.Version != c.OS.Version {
		if t.OS.Version == "" {
			t.OS.Version = c.OS.Version
		} else if c.OS.Version == "" {
			// keep the one we have
		} else {
			return errors.New("cannot merge incompatible os versions: " + t.OS.Version + ", " + c.OS.Version)
		}
	}
	if t.ProfilePattern == "" {
		t.ProfilePattern = c.ProfilePattern
	}
	if t.Description == "" {
		t.Description = c.Description
	}
	if t.Origin == "" {
		t.Origin = c.Origin
	}
	if t.DeviceTemplate == "" {
		t.DeviceTemplate = c.DeviceTemplate
	}
	if t.DeviceOrigin == "" {
		t.DeviceOrigin = c.DeviceOrigin
	}
	if t.SourceFilesystems == nil {
		t.SourceFilesystems = make(map[string]string)
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

func (path HostPath) Resolve(dir string) HostPath {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(string(path)) {
		return path
	}
	return HostPath(filepath.Join(dir, string(path)))
}

func (t *Config) ResolvePaths(dir string) {
	for i, f := range t.Include {
		t.Include[i] = f.Resolve(dir)
	}
	for _, f := range t.Files {
		f.Source = f.Source.Resolve(dir)
	}
	for _, s := range t.Scripts {
		s.File = s.File.Resolve(dir)
	}
	t.SourceConfig = t.SourceConfig.Resolve(dir)
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

func (t *Config) ProfileName(name string) string {
	if t.ProfilePattern != "" {
		pattern := util.Pattern{}
		pattern.SetConstant("container", name)
		profile, err := pattern.Substitute(t.ProfilePattern)
		if err == nil {
			return profile
		}
		fmt.Printf("invalid profile pattern: %s.  Using default.", t.ProfilePattern)
	}
	return name + "." + DefaultProfileSuffix
}

/** Return the filesystem for the given id
In the second argument, return whether the filesystem with the specified id was defined
*/
func (t *Config) FilesystemForId(id string) (*Filesystem, bool) {
	for _, fs := range t.Filesystems {
		if fs.Id == id {
			return fs, true
		}
	}
	return nil, false
}
