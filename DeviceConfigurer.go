package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"melato.org/export/program"
)

type DeviceConfigurer struct {
	Ops            *Ops   `name:""`
	DeviceTemplate string `name:"d" usage:"device to use as template for devices"`
	ProfileDir     string `name:"profile-dir" usage:"directory to save profile files"`
	DryRun         bool   `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	prog           program.Params
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
	if !DirExists(dir) {
		if t.DeviceTemplate != "" && !strings.Contains(t.DeviceTemplate, "@") {
			sname := time.Now().Format("20060102150405")
			t.DeviceTemplate = t.DeviceTemplate + "@" + name + "-" + sname
			err = t.Ops.ZFS().Run("snapshot", t.DeviceTemplate)
			if err != nil {
				return err
			}
		}
		if t.DeviceTemplate == "" {
			err = t.Ops.ZFS().Run("create", fs)
			if err != nil {
				return err
			}
			for _, device := range config.Devices {
				deviceDir := filepath.Join(dir, device.Name)
				if device.Recordsize != "" {
					err := t.Ops.ZFS().Run("create", "-o", "recordsize="+device.Recordsize, filepath.Join(fs, device.Name))
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
			}
		} else {
			err = t.Ops.ZFS().Run("clone", t.DeviceTemplate, fs)
		}
		if err != nil {
			return err
		}
	} else {
		fmt.Println("reusing", dir)
	}
	return t.Ops.ZFS().Run("list", "-r", fs)
}
