package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"

	"strings"

	"melato.org/export/program"
)

type DeviceConfigurer struct {
	Ops        *Ops   `name:""`
	ProfileDir string `name:"profile-dir" usage:"directory to save profile files"`
	DryRun     bool   `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	prog       program.Params
}

func (t *DeviceConfigurer) Init() error {
	t.ProfileDir = "target"
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

func (t *DeviceConfigurer) ConfigureDevices(config *Config, name string) error {
	err := t.CreateDeviceDirs(config, name)
	if err != nil {
		return err
	}
	zfsroot, err := t.Ops.ZFSRoot()
	if err != nil {
		return err
	}
	err = config.CreateProfile(name, t.ProfileDir, zfsroot)
	if err != nil {
		return err
	}
	return err
}

func (t *DeviceConfigurer) CopyTemplate(config *Config, name string) error {
	if config.DeviceTemplate == "" {
		return nil
	}
	var err error
	zfsroot, err := t.Ops.ZFSRoot()
	if err != nil {
		return err
	}
	var templateDir string
	if strings.HasPrefix(config.DeviceTemplate, "/") {
		templateDir = config.DeviceTemplate
	} else {
		templateFS := filepath.Join(zfsroot, config.GetHostFS(), config.DeviceTemplate)
		templateDir = filepath.Join("/", templateFS)
	}
	if !DirExists(templateDir) {
		return errors.New("Device Template does not exist: " + templateDir)
	}
	fs := filepath.Join(zfsroot, config.GetHostFS(), name)
	dir := filepath.Join("/", fs)
	return t.prog.NewProgram("rsync").Sudo(true).Run("-a", templateDir+"/", dir+"/")
}

//			err = t.Ops.ZFS().Run("clone", t.DeviceTemplate, fs)

func (t *DeviceConfigurer) CreateDeviceDirs(config *Config, name string) error {
	if config.Devices == nil {
		return nil
	}
	var err error
	zfsroot, err := t.Ops.ZFSRoot()
	if err != nil {
		return err
	}
	fs := filepath.Join(zfsroot, config.GetHostFS(), name)
	dir := filepath.Join("/", fs)

	if config.DeviceOrigin == "" {
		if !DirExists(dir) {
			err = t.Ops.ZFS().Run("create", fs)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("reusing", dir)
		}
		for _, device := range config.Devices {
			deviceDir := filepath.Join(dir, device.Name)
			if !DirExists(deviceDir) {
				if device.Recordsize != "" || len(device.Zfsproperties) != 0 {
					args := []string{"create"}
					if device.Recordsize != "" {
						args = append(args, "-o", "recordsize="+device.Recordsize)
					}
					for key, value := range device.Zfsproperties {
						args = append(args, "-o", key+"="+value)
					}
					args = append(args, filepath.Join(fs, device.Name))
					err := t.Ops.ZFS().Run(args...)
					if err != nil {
						return err
					}
				} else {
					err = t.prog.NewProgram("mkdir").Sudo(true).Run("-p", deviceDir)
					//err = os.Mkdir(deviceDir, 0755)
					if err != nil {
						return err
					}
				}
				err = t.prog.NewProgram("chown").Sudo(true).Run("-R", "1000000:1000000", deviceDir)
				if err != nil {
					return err
				}
			} else {
				fmt.Println("reusing", deviceDir)
			}
		}
		err = t.CopyTemplate(config, name)
		if err != nil {
			return err
		}
	} else {
		if !DirExists(dir) {
			originFS := filepath.Join(zfsroot, config.GetHostFS(), config.DeviceOrigin)
			err = t.Ops.ZFS().Run("clone", originFS, fs)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("reusing", dir)
		}
	}
	return t.Ops.ZFS().Run("list", "-r", fs)
}
