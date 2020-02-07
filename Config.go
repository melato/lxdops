package lxdops

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"melato.org/export/program"
)

type Config struct {
	OS *OS
	/** Include other configs */
	Includes []string `yaml:"include,omitempty"`
	/** Files or directories that must exist on the host */
	RequiredFiles []string  `yaml:"require,omitempty"`
	HostFS        string    `yaml:"host-fs,omitempty"`
	Devices       []*Device `yaml:"devices,omitempty"`
	Repositories  []string  `yaml:"repositories,omitempty"`
	Profiles      []string  `yaml:"profiles,omitempty"`
	Packages      []string  `yaml:"packages,omitempty"`
	Users         []*User   `yaml:"users,omitempty"`
	Scripts       []*Script `yaml:"scripts,omitempty"`
	Passwords     []string  `yaml:"passwords,omitempty"`
}

type OS struct {
	Name    string `yaml:"name,omitempty" xml:"name,attr,omitempty"`
	Version string `yaml:"version,omitempty" xml:"version,attr,omitempty"`
	osType  OSType `xml:"-"`
}

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
func (t *Config) VerifyFileExists(file string) bool {
	if file != "" && !FileExists(file) {
		fmt.Println("file does not exist: " + file)
		return false
	}
	return true
}

type Device struct {
	Path       string `xml:"path,attr"`
	Name       string `xml:"name,attr"`
	Recordsize string `xml:"recordsize,attr" yaml:",omitempty"`
}

type Script struct {
	/** A name that identifies the script. */
	Name string `xml:"name,attr" yaml:"name"`

	/** The file to run.  This is a file on the host, that is copied to the container in /root/ and run there.
	  Must be an absolute path or a path relative to the ZFS root (the parent of the default storage pool's zfs pool)
	*/
	File string `xml:"file,attr" yaml:"file,omitempty"`
	/** If true, run before packages and users, otherwise after packages and users. */
	First bool `yaml:"first,omitempty"`
	/** Reboot after running this script */
	Reboot bool `xml:"reboot,attr" yaml:"reboot,omitempty"`
	/** The content of the script. */
	Body string `xml:",cdata" yaml:"body,omitempty"`
	/** The directory to run the script in. */
	Dir string `xml:"dir,attr" yaml:"dir,omitempty"`
	/** The uid to run as */
	Uid int `xml:"uid,attr" yaml:"uid,omitempty"`
	/** The gid to run as */
	Gid int `xml:"gid,attr" yaml:"gid,omitempty"`
}

type User struct {
	/** Current - Use the name and uid of the user that is running this program */
	Current bool     `xml:"current,attr" yaml:"current"`
	Name    string   `xml:"name,attr" yaml:"name"`
	Uid     string   `xml:"uid,attr" yaml:"uid,omitempty"`
	Sudo    bool     `xml:"sudo,attr" yaml:"sudo,omitempty"`
	Ssh     bool     `xml:"ssh,attr" yaml:"ssh,omitempty"`
	Shell   string   `xml:"shell,attr" yaml:"shell,omitempty"`
	Home    string   `xml:"home,attr" yaml:"home,omitempty"`
	Groups  []string `xml:"group" yaml:"groups,omitempty"`
}

func (u *User) EffectiveUser() *User {
	if u.Current {
		currentUser, err := user.Current()
		if err == nil {
			var u2 User
			u2 = *u
			u2.Name = currentUser.Username
			u2.Uid = currentUser.Uid
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

func (config *Config) Verify() bool {
	valid := true
	for _, u := range config.Users {
		if !u.IsValidName() {
			valid = false
			fmt.Println("invalid user name: " + u.Name)
		}
	}
	for _, file := range config.RequiredFiles {
		if !config.VerifyFileExists(file) {
			valid = false
		}
	}
	return valid
}

func (u *User) IsValidName() bool {
	if u.Current {
		return true
	}
	re := regexp.MustCompile("^[A-Za-z][A-Za-z0-9_]+$")
	return re.MatchString(u.Name)
}

func (t *Config) GetHostFS() string {
	if t.HostFS != "" {
		return t.HostFS
	}
	return "host"
}

func (t *Config) Print() error {
	return PrintConfigYaml(t)
}

/** Read config without includes */
func ReadRawConfig(file string) (*Config, error) {
	if strings.HasSuffix(file, ".xml") {
		return ReadConfigXml(file)
	}
	return ReadConfigYaml(file)
}

func (config *Config) ProfileName(name string) string {
	return name + ".host"
}

func (config *Config) CreateProfile(name string, profileDir string, zfsRoot string) error {
	err := os.Mkdir(profileDir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	dir := filepath.Join("/", zfsRoot, config.GetHostFS(), name)

	var lines []string

	lines = append(lines, "config: {}")
	lines = append(lines, "description: container disk devices")
	lines = append(lines, "devices:")

	profileName := config.ProfileName(name)
	err = program.NewProgram("lxc").Run("profile", "create", profileName)
	if err != nil {
		return err
	}

	for _, device := range config.Devices {
		// lxc profile device add a1.host etc disk source=/z/host/a1/etc path=/etc/opt
		err := program.NewProgram("lxc").Run("profile", "device", "add", profileName,
			device.Name,
			"disk",
			"path="+device.Path,
			"source="+filepath.Join(dir, device.Name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Config) Merge(c *Config) error {
	//fmt.Printf("Merge %p %p\n", t, c)
	if t.OS == nil {
		t.OS = c.OS
	} else if c.OS == nil {
		// keep the one we have
	} else if t.OS.Name != c.OS.Name {
		return errors.New("cannot merge incompatible oses: " + t.OS.Name + ", " + c.OS.Name)
	} else if t.OS.Version != c.OS.Version {
		if t.OS.Version == "" {
			t.OS.Version = c.OS.Version
		} else {
			return errors.New("cannot merge incompatible os versions: " + t.OS.Version + ", " + c.OS.Version)
		}
	}
	t.RequiredFiles = append(t.RequiredFiles, c.RequiredFiles...)
	t.Devices = append(t.Devices, c.Devices...)
	t.Repositories = append(t.Repositories, c.Repositories...)
	t.Packages = append(t.Packages, c.Packages...)
	t.Profiles = append(t.Profiles, c.Profiles...)
	t.Users = append(t.Users, c.Users...)
	t.Scripts = append(t.Scripts, c.Scripts...)
	t.Passwords = append(t.Passwords, c.Passwords...)

	t.removeDuplicates()
	return nil
}

func (t *Config) removeDuplicates() {
	// remove duplicate strings
	t.Repositories = RemoveDuplicates(t.Repositories)
	t.Packages = RemoveDuplicates(t.Packages)
	t.Passwords = RemoveDuplicates(t.Passwords)
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
		fmt.Println(file)
		c, err := ReadConfig(file)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			result = c
		} else {
			fmt.Printf("before merge %p %p %d\n", result, c, len(result.Devices))
			result.Merge(c)
			fmt.Printf("after merge %p %p %d\n", result, c, len(result.Devices))
		}
	}
	return result, nil
}

func (t *Config) merge(file string, included map[string]bool) error {
	if _, found := included[file]; found {
		return errors.New("include loop, file=" + file)
	}
	included[file] = true
	config, err := ReadRawConfig(file)
	if err != nil {
		return err
	}
	for _, file := range config.Includes {
		err := t.merge(file, included)
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
