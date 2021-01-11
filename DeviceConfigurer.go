package lxdops

import (
	"errors"
	"path/filepath"

	"strings"

	"melato.org/export/program"
)

type DeviceConfigurer struct {
	Ops  *Ops
	prog program.Params
}

func NewDeviceConfigurer(ops *Ops) *DeviceConfigurer {
	t := &DeviceConfigurer{Ops: ops}
	t.prog.Trace = t.Ops.Trace
	return t
}

func (t *DeviceConfigurer) SetDryRun(dryRun bool) {
	t.prog.DryRun = dryRun
}

func ProfileExists(profile string) bool {
	// Not sure what profile get does, but it returns an error if the profile doesn't exist.
	// "x" is a key.  It doesn't matter what key we use for our purpose.
	err := program.NewProgram("lxc").Run("profile", "get", profile, "x")
	return err == nil
}

type PatternInfo struct {
	Configurer *DeviceConfigurer
	Config     *Config
	Container  string
}

func (t *PatternInfo) Get(key string) (string, error) {
	if strings.HasPrefix(key, ".") {
		pkey := key[1:]
		value, found := t.Config.Properties[pkey]
		if found {
			return value, nil
		}
		return "", errors.New("property not found: " + pkey)
	}
	if key == "container" {
		return t.Container, nil
	}
	if key == "zfsroot" {
		zfsroot, err := t.Configurer.Ops.ZFSRoot()
		if err != nil {
			return "", nil
		}
		return zfsroot, err
	}
	return "", errors.New("unknown key: " + key)
}

func (t *PatternInfo) Substitute(pattern string) (string, error) {
	if strings.IndexAny(pattern, "{}") >= 0 {
		return "", errors.New(`pattern contains {}, please replace with (): ` + pattern)
	}
	return Substitute(pattern, t.Get)
}

func (t *DeviceConfigurer) CreateFilesystem(config *Config, fs *Filesystem, name string) error {
	pattern := &PatternInfo{Configurer: t, Config: config, Container: name}
	path, err := pattern.Substitute(fs.Pattern)
	if err != nil {
		return err
	}
	if strings.HasPrefix(path, "/") {
		return t.CreateDir(path, false)
	} else {
		if config.DeviceOrigin == "" {
			args := []string{"create", "-p"}
			for key, value := range fs.Zfsproperties {
				args = append(args, "-o", key+"="+value)
			}
			args = append(args, path)
			err := t.Ops.ZFS().Run(args...)
			if err != nil {
				return err
			}
		} else {
			parts := strings.Split(config.DeviceOrigin, "@")
			if len(parts) != 2 {
				return errors.New("device origin should be a snapshot: " + config.DeviceOrigin)
			}
			originPattern := &PatternInfo{Configurer: t, Config: config, Container: parts[0]}
			originDataset, err := originPattern.Substitute(fs.Pattern)
			if err != nil {
				return err
			}
			err = t.Ops.ZFS().Run("clone", "-p", originDataset+"@"+parts[1], path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *DeviceConfigurer) CreateDir(dir string, chown bool) error {
	if !DirExists(dir) {
		err := t.prog.NewProgram("mkdir").Sudo(true).Run("-p", dir)
		//err = os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
		if chown {
			err = t.prog.NewProgram("chown").Sudo(true).Run("-R", "1000000:1000000", dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *DeviceConfigurer) FilesystemPaths(config *Config, name string) (map[string]string, error) {
	pattern := &PatternInfo{Configurer: t, Config: config, Container: name}
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

func (t *DeviceConfigurer) DeviceDir(config *Config, filesystems map[string]string, device *Device, name string) (string, error) {
	pattern := &PatternInfo{Configurer: t, Config: config, Container: name}
	dir, err := pattern.Substitute(device.Dir)
	if err != nil {
		return "", err
	}
	if device.Filesystem != "" {
		fs, found := filesystems[device.Filesystem]
		if !found {
			return "", errors.New("missing filesystem: " + device.Filesystem)
		}
		dir = filepath.Join(fs, dir)
	}
	return dir, nil
}

func (t *DeviceConfigurer) ConfigureDevices(config *Config, name string) error {
	filesystems, err := t.FilesystemPaths(config, name)
	if err != nil {
		return err
	}
	for _, fs := range config.Filesystems {
		fsDir, _ := filesystems[fs.Id]
		if !DirExists(fsDir) {
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
	for _, device := range config.Devices {
		if profileName == "" {
			profileName = config.ProfileName(name)
			if !ProfileExists(profileName) {
				useProfile = true
				err := program.NewProgram("lxc").Run("profile", "create", profileName)
				if err != nil {
					return err
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
			if !DirExists(templateDir) {
				return errors.New("Device Template does not exist: " + templateDir)
			}
			err = t.prog.NewProgram("rsync").Sudo(true).Run("-a", templateDir+"/", dir+"/")
			if err != nil {
				return err
			}
		}
		// lxc profile device add a1.devices etc disk source=/z/host/a1/etc path=/etc/opt
		if useProfile {
			err := program.NewProgram("lxc").Run("profile", "device", "add", profileName,
				device.Name,
				"disk",
				"path="+device.Path,
				"source="+dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
