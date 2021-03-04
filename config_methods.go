package lxdops

import (
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

func (t *Config) ProfileName(name string) string {
	if t.Profile != "" {
		properties := &util.PatternProperties{}
		properties.SetConstant("instance", name)
		profile, err := t.Profile.Substitute(properties)
		if err == nil {
			return profile
		}
		fmt.Printf("invalid profile pattern: %s.  Using default.", t.Profile)
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

func (t *Config) GetSourceConfig() (*Config, error) {
	if t.SourceConfig == "" {
		return nil, nil
	}
	if t.sourceConfig == nil {
		config, err := ReadConfig(string(t.SourceConfig))
		if err != nil {
			return nil, err
		}
		t.sourceConfig = config
	}
	return t.sourceConfig, nil
}

func (config *Config) NewProperties(name string) *util.PatternProperties {
	properties := &util.PatternProperties{Properties: config.Properties}
	properties.SetConstant("instance", name)
	project := config.Project
	var projectSlash, project_instance string
	if project == "" || project == "default" {
		project = "default"
		projectSlash = ""
		project_instance = name
	} else {
		projectSlash = project + "/"
		project_instance = project + "_" + name
	}
	properties.SetConstant("project", project)
	properties.SetConstant("project/", projectSlash)
	properties.SetConstant("project_instance", project_instance)
	return properties
}
