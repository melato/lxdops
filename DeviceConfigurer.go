package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"

	"strings"

	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type DeviceConfigurer struct {
	Config  *Config
	Trace   bool
	DryRun  bool
	FuncMap map[string]func() (string, error)
}

func NewDeviceConfigurer(config *Config) *DeviceConfigurer {
	return &DeviceConfigurer{Config: config}
}

func (t *DeviceConfigurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace, DryRun: t.DryRun}
}

func (t *DeviceConfigurer) AddFuncs(map[string]func() (string, error)) {
}

func (t *DeviceConfigurer) NewPattern(name string) *util.Pattern {
	pattern := &util.Pattern{Properties: t.Config.Properties}
	pattern.SetConstant("container", name)
	pattern.SetFunction("zfsroot", func() (string, error) {
		return ZFSRoot()
	})
	return pattern
}

func (t *DeviceConfigurer) CreateFilesystem(fs *Filesystem, name string) error {
	pattern := t.NewPattern(name)
	path, err := pattern.Substitute(fs.Pattern)
	if err != nil {
		return err
	}
	script := t.NewScript()
	if strings.HasPrefix(path, "/") {
		return t.CreateDir(path, true)
	} else {
		if t.Config.DeviceOrigin == "" {
			args := []string{"zfs", "create", "-p"}
			for key, value := range fs.Zfsproperties {
				args = append(args, "-o", key+"="+value)
			}
			args = append(args, path)
			script.Run("sudo", args...)
			t.chownDir(script, filepath.Join("/", path))
		} else {
			parts := strings.Split(t.Config.DeviceOrigin, "@")
			if len(parts) != 2 {
				return errors.New("device origin should be a snapshot: " + t.Config.DeviceOrigin)
			}
			originPattern := t.NewPattern(parts[0])
			originDataset, err := originPattern.Substitute(fs.Pattern)
			if err != nil {
				return err
			}
			script.Run("sudo", "zfs", "clone", "-p", originDataset+"@"+parts[1], path)
		}
	}
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

func (t *DeviceConfigurer) FilesystemPaths(name string) (map[string]string, error) {
	pattern := t.NewPattern(name)
	filesystems := make(map[string]string)
	for _, fs := range t.Config.Filesystems {
		path, err := pattern.Substitute(fs.Pattern)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(path, "/") {
			path = filepath.Join("/", path)
		}
		filesystems[fs.Id] = path
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

func (t *DeviceConfigurer) DeviceDir(filesystems map[string]string, device *Device, name string) (string, error) {
	pattern := t.NewPattern(name)
	var fsDir, dir string
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
		fsDir = filesystems[fs.Id]
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
		return filepath.Join(fsDir, dir), nil
	} else {
		return fsDir, nil
	}
}

func (t *DeviceConfigurer) ConfigureDevices(name string) error {
	filesystems, err := t.FilesystemPaths(name)
	if err != nil {
		return err
	}
	for _, fs := range FilesystemList(t.Config.Filesystems).Sorted() {
		fsDir, _ := filesystems[fs.Id]
		if t.Config.DeviceOrigin != "" || !util.DirExists(fsDir) {
			err := t.CreateFilesystem(fs, name)
			if err != nil {
				return err
			}
		}
	}
	var templateFilesystems map[string]string
	if t.Config.DeviceTemplate != "" {
		templateFilesystems, err = t.FilesystemPaths(t.Config.DeviceTemplate)
		if err != nil {
			return err
		}
	}
	var profileName string
	var useProfile bool
	script := t.NewScript()
	for _, device := range t.Config.Devices {
		if profileName == "" {
			profileName = t.Config.ProfileName(name)
			if !ProfileExists(profileName) {
				useProfile = true
				script.Run("lxc", "profile", "create", profileName)
				if script.HasError() {
					return script.Error()
				}
			}
		}
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
		// lxc profile device add a1.devices etc disk source=/z/host/a1/etc path=/etc/opt
		if useProfile {
			script.Run("lxc", "profile", "device", "add", profileName,
				device.Name,
				"disk",
				"path="+device.Path,
				"source="+dir)
		}
		if script.Error() != nil {
			return script.Error()
		}
	}
	return nil
}

func (t *DeviceConfigurer) CreateProfile(name string) error {
	filesystems, err := t.FilesystemPaths(name)
	if err != nil {
		return err
	}
	profileName := t.Config.ProfileName(name)
	s := t.NewScript()
	s.Run("lxc", "profile", "create", profileName)
	for _, device := range t.Config.Devices {
		dir, err := t.DeviceDir(filesystems, device, name)
		if err != nil {
			return err
		}
		s.Run("lxc", "profile", "device", "add", profileName,
			device.Name,
			"disk",
			"path="+device.Path,
			"source="+dir)
	}
	return s.Error()
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

func (t *DeviceConfigurer) ListFilesystems(name string) ([]string, error) {
	filesystems, err := t.FilesystemPaths(name)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, fs := range t.Config.Filesystems {
		fsDir, _ := filesystems[fs.Id]
		if util.DirExists(fsDir) {
			result = append(result, fsDir)
		}
	}
	return result, nil
}

func (t *DeviceConfigurer) PrintFilesystems(name string) error {
	pattern := t.NewPattern(name)
	for _, fs := range t.Config.Filesystems {
		path, err := pattern.Substitute(fs.Pattern)
		if err != nil {
			return err
		}
		fmt.Println(path)
	}
	return nil
}
