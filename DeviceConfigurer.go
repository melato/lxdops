package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"

	"strings"

	"melato.org/lxdops/util"
	"melato.org/script"
)

type DeviceConfigurer struct {
	Ops    *Ops
	Trace  bool
	DryRun bool
}

func NewDeviceConfigurer(ops *Ops) *DeviceConfigurer {
	t := &DeviceConfigurer{Ops: ops, Trace: ops.Trace}
	return t
}

func (t *DeviceConfigurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Ops.Trace, DryRun: t.DryRun}
}

func (t *DeviceConfigurer) SetDryRun(dryRun bool) {
	t.DryRun = dryRun
}

func ProfileExists(profile string) bool {
	// Not sure what profile get does, but it returns an error if the profile doesn't exist.
	// "x" is a key.  It doesn't matter what key we use for our purpose.
	script := script.Script{}
	return script.Cmd("lxc", "profile", "get", profile, "x").MergeStderr().ToNull()
}

func (t *DeviceConfigurer) NewPattern(config *Config, name string) *util.Pattern {
	pattern := &util.Pattern{Properties: config.Properties}
	pattern.SetConstant("container", name)
	pattern.SetFunction("zfsroot", func() (string, error) {
		return t.Ops.ZFSRoot()
	})
	return pattern
}

func (t *DeviceConfigurer) CreateFilesystem(config *Config, fs *Filesystem, name string) error {
	pattern := t.NewPattern(config, name)
	path, err := pattern.Substitute(fs.Pattern)
	if err != nil {
		return err
	}
	script := t.NewScript()
	if strings.HasPrefix(path, "/") {
		return t.CreateDir(path, true)
	} else {
		if config.DeviceOrigin == "" {
			args := []string{"zfs", "create", "-p"}
			for key, value := range fs.Zfsproperties {
				args = append(args, "-o", key+"="+value)
			}
			args = append(args, path)
			script.Run("sudo", args...)
		} else {
			parts := strings.Split(config.DeviceOrigin, "@")
			if len(parts) != 2 {
				return errors.New("device origin should be a snapshot: " + config.DeviceOrigin)
			}
			originPattern := t.NewPattern(config, parts[0])
			originDataset, err := originPattern.Substitute(fs.Pattern)
			if err != nil {
				return err
			}
			script.Run("sudo", "zfs", "clone", "-p", originDataset+"@"+parts[1], path)
		}
		t.chownDir(script, filepath.Join("/", path))
	}
	return script.Error
}

func (t *DeviceConfigurer) chownDir(scr *script.Script, dir string) {
	scr.Run("sudo", "chown", "-R", "1000000:1000000", dir)
}

func (t *DeviceConfigurer) CreateDir(dir string, chown bool) error {
	if !util.DirExists(dir) {
		script := t.NewScript()
		script.Run("sudo", "mkdir", "-p", dir)
		//err = os.Mkdir(dir, 0755)
		if chown {
			t.chownDir(script, dir)
		}
		return script.Error
	}
	return nil
}

func (t *DeviceConfigurer) FilesystemPaths(config *Config, name string) (map[string]string, error) {
	pattern := t.NewPattern(config, name)
	filesystems := make(map[string]string)
	for _, fs := range config.Filesystems {
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

func (t *DeviceConfigurer) DeviceFilesystem(config *Config, device *Device) (*Filesystem, error) {
	for _, fs := range config.Filesystems {
		if fs.Id == device.Filesystem {
			return fs, nil
		}
	}
	return nil, errors.New("no such filesystem: " + device.Filesystem)
}

func (t *DeviceConfigurer) DeviceDir(config *Config, filesystems map[string]string, device *Device, name string) (string, error) {
	pattern := t.NewPattern(config, name)
	var fsDir, dir string
	var substituteDir bool
	var err error
	if strings.HasPrefix(device.Dir, "/") {
		dir = device.Dir
		substituteDir = true
	} else {
		fs, err := t.DeviceFilesystem(config, device)
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

func (t *DeviceConfigurer) ConfigureDevices(config *Config, name string) error {
	filesystems, err := t.FilesystemPaths(config, name)
	if err != nil {
		return err
	}
	for _, fs := range config.Filesystems {
		fsDir, _ := filesystems[fs.Id]
		if !util.DirExists(fsDir) {
			err := t.CreateFilesystem(config, fs, name)
			if err != nil {
				return err
			}
		}
	}
	var templateFilesystems map[string]string
	if config.DeviceTemplate != "" {
		templateFilesystems, err = t.FilesystemPaths(config, config.DeviceTemplate)
		if err != nil {
			return err
		}
	}
	var profileName string
	var useProfile bool
	script := t.NewScript()
	for _, device := range config.Devices {
		if profileName == "" {
			profileName = config.ProfileName(name)
			if !ProfileExists(profileName) {
				useProfile = true
				script.Run("lxc", "profile", "create", profileName)
				if script.Error != nil {
					return script.Error
				}
			}
		}
		dir, err := t.DeviceDir(config, filesystems, device, name)
		if err != nil {
			return err
		}
		err = t.CreateDir(dir, true)
		if err != nil {
			return err
		}
		if config.DeviceTemplate != "" {
			templateDir, err := t.DeviceDir(config, templateFilesystems, device, config.DeviceTemplate)
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
		if script.Error != nil {
			return script.Error
		}
	}
	return nil
}

func (t *DeviceConfigurer) ListFilesystems(config *Config, name string) ([]string, error) {
	filesystems, err := t.FilesystemPaths(config, name)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, fs := range config.Filesystems {
		fsDir, _ := filesystems[fs.Id]
		if util.DirExists(fsDir) {
			result = append(result, fsDir)
		}
	}
	return result, nil
}
