package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"

	"strings"

	"melato.org/export/program"
)

type DeviceConfigurer struct {
	Ops    *Ops `name:""`
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	prog   program.Params
}

func (t *DeviceConfigurer) Init() error {
	return nil
}

func (t *DeviceConfigurer) Configured() error {
	t.prog.DryRun = t.DryRun
	t.prog.Trace = t.Ops.Trace
	return nil
}

func (t *DeviceConfigurer) Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: {profile-name} {configfile}...")
	}
	name := args[0]
	var err error
	var config *Config
	config, err = ReadConfigs(args[1:]...)
	if err != nil {
		return err
	}
	if !config.Verify() {
		return errors.New("prerequisites not met")
	}
	return t.ConfigureDevices(config, name)
}

func ProfileExists(profile string) bool {
	// Not sure what profile get does, but it returns an error if the profile doesn't exist.
	// "x" is a key.  It doesn't matter what key we use for our purpose.
	err := program.NewProgram("lxc").Run("profile", "get", profile, "x")
	return err == nil
}

type DeviceInfo struct {
	Configurer *DeviceConfigurer
	Config     *Config
	Device     *Device
	Container  string
	Dataset    string
	Dir        string
}

func (t *DeviceInfo) init() error {
	var err error
	t.Dataset, err = t.Substitute(t.Device.Dataset)
	if err != nil {
		return err
	}
	dir, err := t.Substitute(t.Device.Dir)
	if err != nil {
		return err
	}
	if t.Dataset == "" && dir == "" {
		t.Dataset, err = t.Substitute("(.host)/(.container)")
		if err != nil {
			return err
		}
		dir = t.Device.Name

	}
	t.Dir = filepath.Join("/", t.Dataset, dir)
	return nil
}

func (t *DeviceConfigurer) NewDeviceInfo(config *Config, device *Device, container string) (*DeviceInfo, error) {
	info := &DeviceInfo{Configurer: t, Config: config, Container: container, Device: device}
	err := info.init()
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (t *DeviceInfo) CreateDataset(isNewDataset bool) error {
	if t.Dataset == "" {
		return nil
	}
	if t.Config.DeviceOrigin == "" {
		args := []string{"create", "-p"}
		if t.Device.Recordsize != "" {
			args = append(args, "-o", "recordsize="+t.Device.Recordsize)
		}
		for key, value := range t.Device.Zfsproperties {
			args = append(args, "-o", key+"="+value)
		}
		args = append(args, t.Dataset)
		err := t.Configurer.Ops.ZFS().Run(args...)
		if err != nil {
			return err
		}
	} else if isNewDataset {
		parts := strings.Split(t.Config.DeviceOrigin, "@")
		if len(parts) != 2 {
			return errors.New("device origin should be a snapshot: " + t.Config.DeviceOrigin)
		}
		originInfo, err := t.Configurer.NewDeviceInfo(t.Config, t.Device, parts[0])
		if err != nil {
			return err
		}
		err = t.Configurer.Ops.ZFS().Run("clone", "-p", originInfo.Dataset+"@"+parts[1], t.Dataset)
		if err != nil {
			return err
		}
	}
	return t.Configurer.Ops.ZFS().Run("list", "-r", t.Dataset)
}

func (t *DeviceInfo) Create(isNewDataset bool) error {
	if !DirExists(t.Dir) {
		err := t.CreateDataset(isNewDataset)
		if err != nil {
			return err
		}
		chown := false
		if !DirExists(t.Dir) {
			err = t.Configurer.prog.NewProgram("mkdir").Sudo(true).Run("-p", t.Dir)
			//err = os.Mkdir(t.Dir, 0755)
			if err != nil {
				return err
			}
			chown = true
		} else if t.Config.DeviceOrigin == "" {
			// the device directory is the dataset directory we just created, so change the ownership.
			chown = true
		}
		// do not change the ownership of a directory that was cloned
		if chown {
			err = t.Configurer.prog.NewProgram("chown").Sudo(true).Run("-R", "1000000:1000000", t.Dir)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Println("reusing", t.Dir)
	}
	return nil
}

func (t *DeviceInfo) Get(key string) (string, bool) {
	if !strings.HasPrefix(key, ".") {
		value, found := t.Config.Properties[key]
		return value, found
	}
	if key == ".container" {
		return t.Container, true
	}
	if key == ".zfsroot" {
		zfsroot, err := t.Configurer.Ops.ZFSRoot()
		if err != nil {
			return "", false
		}
		return zfsroot, true
	}
	if key == ".host" {
		zfsroot, err := t.Configurer.Ops.ZFSRoot()
		if err != nil {
			return "", false
		}
		return filepath.Join(zfsroot, t.Config.GetHostFS()), true
	}
	return "", false
}

func (t *DeviceInfo) Substitute(pattern string) (string, error) {
	return Substitute(pattern, t.Get)
}

func (t *DeviceConfigurer) ConfigureDevices(config *Config, name string) error {
	var profileName string
	var useProfile bool
	datasets := make(map[string]bool)
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
		info, err := t.NewDeviceInfo(config, device, name)
		if err != nil {
			return err
		}
		isNew := true
		if datasets[info.Dataset] {
			isNew = false
		} else {
			datasets[info.Dataset] = true
		}
		err = info.Create(isNew)
		if err != nil {
			return err
		}
		if config.DeviceTemplate != "" {
			templateInfo, err := t.NewDeviceInfo(config, device, config.DeviceTemplate)
			if err != nil {
				return err
			}
			if !DirExists(templateInfo.Dir) {
				return errors.New("Device Template does not exist: " + templateInfo.Dir)
			}
			err = t.prog.NewProgram("rsync").Sudo(true).Run("-a", templateInfo.Dir+"/", info.Dir+"/")
			if err != nil {
				return err
			}
		}
		// lxc profile device add a1.host etc disk source=/z/host/a1/etc path=/etc/opt
		if useProfile {
			err := program.NewProgram("lxc").Run("profile", "device", "add", profileName,
				device.Name,
				"disk",
				"path="+device.Path,
				"source="+info.Dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
