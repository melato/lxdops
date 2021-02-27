package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"

	"strings"

	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type DeviceConfigurer struct {
	Client  *LxdClient
	Config  *Config
	Trace   bool
	DryRun  bool
	FuncMap map[string]func() (string, error)

	sourceFilesystems map[string]string
}

func NewDeviceConfigurer(client *LxdClient, config *Config) *DeviceConfigurer {
	return &DeviceConfigurer{Client: client, Config: config}
}

func (t *DeviceConfigurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace, DryRun: t.DryRun}
}

func (t *DeviceConfigurer) NewPattern(name string) *util.Pattern {
	pattern := &util.Pattern{}
	pattern.SetConstant("container", name)
	pattern.SetFunction("zfsroot", func() (string, error) {
		dataset, err := t.Client.GetDefaultDataset()
		if err != nil {
			return "", err
		}
		root := filepath.Dir(dataset)
		if root == "" {
			return "", errors.New("cannot determine zfsroot for dataset: " + dataset)
		}
		return root, nil
	})
	return pattern
}

func (t *DeviceConfigurer) CreateFilesystem(fs *Filesystem, name string) error {
	pattern := t.NewPattern(name)
	path, err := pattern.Substitute(fs.Pattern)
	if err != nil {
		return err
	}
	if strings.HasPrefix(path, "/") {
		return t.CreateDir(path, true)
	}
	script := t.NewScript()
	if t.Config.DeviceOrigin != "" {
		parts := strings.Split(t.Config.DeviceOrigin, "@")
		if len(parts) != 2 {
			return errors.New("device origin should be a snapshot: " + t.Config.DeviceOrigin)
		}
		sourceInstance, sourceSnapshot := parts[0], parts[1]
		sourcePattern := t.NewPattern(sourceInstance)
		fsPattern, exists := t.sourceFilesystems[fs.Id]
		if exists {
			// clone
			sourceDataset, err := sourcePattern.Substitute(fsPattern)
			if err != nil {
				return err
			}
			script.Run("sudo", "zfs", "clone", "-p", sourceDataset+"@"+sourceSnapshot, path)
			return script.Error()
		}
	}

	// create
	args := []string{"zfs", "create", "-p"}
	for key, value := range fs.Zfsproperties {
		args = append(args, "-o", key+"="+value)
	}
	args = append(args, path)
	script.Run("sudo", args...)
	t.chownDir(script, filepath.Join("/", path))
	return script.Error()
}

func (t *DeviceConfigurer) chownDir(scr *script.Script, dir string) {
	scr.Run("sudo", "chown", "1000000:1000000", dir)
}

func (t *DeviceConfigurer) CreateDir(dir string, chown bool) error {
	if !util.DirExists(dir) {
		script := t.NewScript()
		script.Run("sudo", "mkdir", "-p", dir)
		//err = os.Mkdir(dir, 0755)
		if chown {
			t.chownDir(script, dir)
		}
		return script.Error()
	}
	return nil
}

type FSPath string

func (path FSPath) Dir() string {
	if strings.HasPrefix(string(path), "/") {
		return string(path)
	} else {
		return "/" + string(path)
	}
}

func (t *DeviceConfigurer) FilesystemPaths(name string, overrides map[string]string) (map[string]FSPath, error) {
	pattern := t.NewPattern(name)
	filesystems := make(map[string]FSPath)
	for _, fs := range t.Config.Filesystems {
		fsPattern, overriden := overrides[fs.Id]
		if !overriden {
			fsPattern = fs.Pattern
		}
		path, err := pattern.Substitute(fsPattern)
		if err != nil {
			return nil, err
		}
		filesystems[fs.Id] = FSPath(path)
	}
	return filesystems, nil
}

func (t *DeviceConfigurer) DeviceFilesystem(device *Device) (*Filesystem, error) {
	for _, fs := range t.Config.Filesystems {
		if fs.Id == device.Filesystem {
			return fs, nil
		}
	}
	return nil, errors.New("no such filesystem: " + device.Filesystem)
}

func (t *DeviceConfigurer) DeviceDir(filesystems map[string]FSPath, device *Device, name string) (string, error) {
	pattern := t.NewPattern(name)
	var fsPath FSPath
	var dir string
	var substituteDir bool
	var err error
	if strings.HasPrefix(device.Dir, "/") {
		dir = device.Dir
		substituteDir = true
	} else {
		fs, err := t.DeviceFilesystem(device)
		if err != nil {
			return "", err
		}
		fsPath = filesystems[fs.Id]
		if device.Dir == "" {
			dir = device.Name
		} else if device.Dir == "." {
			dir = ""
		} else {
			dir = device.Dir
			substituteDir = true
		}
	}

	if substituteDir {
		dir, err = pattern.Substitute(device.Dir)
		if err != nil {
			return "", err
		}
	}
	if dir != "" {
		return filepath.Join(fsPath.Dir(), dir), nil
	} else {
		return fsPath.Dir(), nil
	}
}

func (t *DeviceConfigurer) initSourceFilesystems() error {
	var config *Config
	var err error
	if t.Config.SourceConfig != "" {
		config, err = ReadConfigs(string(t.Config.SourceConfig))
		if err != nil {
			return err
		}
	} else {
		config = t.Config
	}
	t.sourceFilesystems = make(map[string]string)
	for _, fs := range config.Filesystems {
		t.sourceFilesystems[fs.Id] = fs.Pattern
	}
	// use t.Config.SourceFilesystem, but don't use SourceConfig SourceFilesystems
	for id, pattern := range t.Config.SourceFilesystems {
		t.sourceFilesystems[id] = pattern
	}

	return nil
}

func (t *DeviceConfigurer) ConfigureDevices(name string) error {
	err := t.initSourceFilesystems()
	if err != nil {
		return err
	}

	filesystems, err := t.FilesystemPaths(name, nil)
	if err != nil {
		return err
	}
	for _, fs := range FilesystemList(t.Config.Filesystems).Sorted() {
		fsPath, _ := filesystems[fs.Id]
		if t.Config.DeviceOrigin != "" || !util.DirExists(fsPath.Dir()) {
			err := t.CreateFilesystem(fs, name)
			if err != nil {
				return err
			}
		}
	}
	var templateFilesystems map[string]FSPath
	if t.Config.DeviceTemplate != "" {
		templateFilesystems, err = t.FilesystemPaths(t.Config.DeviceTemplate, t.sourceFilesystems)
		if err != nil {
			return err
		}
	}
	script := t.NewScript()
	for _, device := range t.Config.Devices {
		dir, err := t.DeviceDir(filesystems, device, name)
		if err != nil {
			return err
		}
		if t.Config.DeviceOrigin == "" {
			err = t.CreateDir(dir, true)
			if err != nil {
				return err
			}
		}
		if t.Config.DeviceTemplate != "" {
			templateDir, err := t.DeviceDir(templateFilesystems, device, t.Config.DeviceTemplate)
			if err != nil {
				return err
			}
			if util.DirExists(templateDir) {
				script.Run("sudo", "rsync", "-a", templateDir+"/", dir+"/")
			} else {
				fmt.Println("skipping missing Device Template: " + templateDir)
			}
		}
		if script.Error() != nil {
			return script.Error()
		}
	}
	return nil
}

func (t *DeviceConfigurer) CreateProfile(name string) error {
	filesystems, err := t.FilesystemPaths(name, nil)
	if err != nil {
		return err
	}
	devices := make(map[string]map[string]string)

	for _, device := range t.Config.Devices {
		dir, err := t.DeviceDir(filesystems, device, name)
		if err != nil {
			return err
		}
		devices[device.Name] = map[string]string{"type": "disk", "path": device.Path, "source": dir}
	}
	profileName := t.Config.ProfileName(name)
	server, _, err := t.Client.ContainerServer(name)
	if err != nil {
		return err
	}
	post := api.ProfilesPost{Name: profileName, ProfilePut: api.ProfilePut{Devices: devices, Description: "lxdops devices"}}
	if t.Trace {
		fmt.Printf("create profile %s:\n", profileName)
		util.PrintYaml(&post)
	}
	if !t.DryRun {
		return server.CreateProfile(post)
	}
	return nil
}

func (t *DeviceConfigurer) RenameFilesystems(oldname, newname string) error {
	oldpattern := t.NewPattern(oldname)
	newpattern := t.NewPattern(newname)
	s := t.NewScript()
	for _, fs := range RootFilesystems(t.Config.Filesystems) {
		oldpath, err := oldpattern.Substitute(fs.Pattern)
		if err != nil {
			return err
		}
		newpath, err := newpattern.Substitute(fs.Pattern)
		if err != nil {
			return err
		}
		if strings.HasPrefix(oldpath, "/") {
			if util.DirExists(newpath) {
				return errors.New(newpath + ": already exists")
			}
			s.Run("mv", oldpath, newpath)
		} else {
			s.Run("sudo", "zfs", "rename", oldpath, newpath)
		}
	}
	return s.Error()
}

func (t *DeviceConfigurer) ListFilesystems(name string) ([]FSPath, error) {
	filesystems, err := t.FilesystemPaths(name, nil)
	if err != nil {
		return nil, err
	}
	var result []FSPath
	for _, fs := range t.Config.Filesystems {
		fsPath, _ := filesystems[fs.Id]
		if util.DirExists(fsPath.Dir()) {
			result = append(result, fsPath)
		}
	}
	return result, nil
}

func (t *DeviceConfigurer) PrintFilesystems(name string) error {
	pattern := t.NewPattern(name)
	fmt.Printf("%s: %s -> %s\n", "id", "pattern", "path")
	for _, fs := range t.Config.Filesystems {
		path, err := pattern.Substitute(fs.Pattern)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %s -> %s\n", fs.Id, fs.Pattern, path)
	}
	return nil
}
