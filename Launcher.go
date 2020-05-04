package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"melato.org/export/program"
)

type Launcher struct {
	Ops               *Ops   `name:""`
	ContainerTemplate string `name:"c" usage:"container to use as template"`
	DeviceTemplate    string `name:"d" usage:"device to use as template for devices"`
	ProfileDir        string `name:"profile-dir" usage:"directory to save profile files"`
	DryRun            bool   `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	prog              program.Params
}

func (t *Launcher) Init() error {
	t.ProfileDir = "target"
	return nil
}

func (t *Launcher) Configured() error {
	t.prog.DryRun = t.DryRun
	t.prog.Trace = t.Ops.Trace
	return nil
}

func (op *Launcher) Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: {name} {configfile}...")
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
	return op.LaunchContainer(config, name)
}

func (t *Launcher) NewConfigurer() *Configurer {
	var c = &Configurer{ops: t.Ops, DryRun: t.DryRun, prog: t.prog}
	c.Packages = true
	c.Users = true
	c.Scripts = true
	return c
}

func (t *Launcher) LaunchContainer(config *Config, name string) error {
	var err error
	osType := config.OS.Type()
	if osType == nil {
		return errors.New("unsupported OS type.  must be ubuntu or alpine")
	}

	err = t.CreateDeviceDirs(config, name)
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
	for _, rep := range config.Repositories {
		t.Ops.CloneRepository(rep)
	}
	var profiles []string
	profiles = append(profiles, config.Profiles...)
	//profiles := []string{"default", "dev", "opt", "tools"}
	if config.Devices != nil {
		if len(profiles) == 0 {
			profiles = append(profiles, "default")
		}
		profiles = append(profiles, config.ProfileName(name))
	}
	fmt.Println("profiles", profiles)
	containerTemplate := t.ContainerTemplate
	t.Ops.copyHostInfo()
	if containerTemplate == "" {
		var lxcArgs []string
		lxcArgs = append(lxcArgs, "launch")

		osVersion := config.OS.Version
		if osVersion == "" {
			return errors.New("Missing version")
		}
		if config.OS.IsUbuntu() {
			opt, err := t.Ops.GetPath("opt")
			if err != nil {
				return err
			}
			t.prog.NewProgram("mkdir").Sudo(true).Run("-p", filepath.Join(opt, "ubuntu"))
			lsb, err := ReadProperties("/etc/lsb-release")
			if err != nil {
				return err
			}
			release := lsb["DISTRIB_RELEASE"]
			if release != "" {
				file := filepath.Join(opt, "ubuntu", "ubuntu-"+release+".list")
				err = t.prog.NewProgram("cp").Sudo(true).Run("/etc/apt/sources.list", file)
			}
		}
		lxcArgs = append(lxcArgs, osType.ImageName(osVersion))
		for _, profile := range profiles {
			lxcArgs = append(lxcArgs, "-p", profile)
		}
		lxcArgs = append(lxcArgs, name)
		err = t.prog.NewProgram("lxc").Run(lxcArgs...)
		if err != nil {
			return err
		}
		t.NewConfigurer().ConfigureContainer(config, name)
	} else {
		copyArgs := []string{"copy"}
		if !strings.Contains(containerTemplate, "/") {
			copyArgs = append(copyArgs, "--container-only")
		}
		copyArgs = append(copyArgs, containerTemplate, name)
		err = t.prog.NewProgram("lxc").Run(copyArgs...)
		if err != nil {
			return err
		}
		copyArgs = append(copyArgs, containerTemplate)

		err = t.prog.NewProgram("lxc").Run("profile", "apply", name, strings.Join(profiles, ","))
		if err != nil {
			return err
		}
		err = t.prog.NewProgram("lxc").Run("start", name)
		if err != nil {
			return err
		}
		err = t.Ops.waitForNetwork(name)
		if err != nil {
			return err
		}
	}
	return err
}

func (t *Launcher) CreateDeviceDirs(config *Config, name string) error {
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
