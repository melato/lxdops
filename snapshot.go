package lxdops

import (
	"errors"
	"strings"

	"melato.org/script/v2"
)

type Snapshot struct {
	Client        *LxdClient `name:"-"`
	ConfigOptions ConfigOptions
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Destroy       bool `name:"d" usage:"destroy snapshots"`
	Recursive     bool `name:"R" usage:"zfs destroy -R: Recursively destroy all dependents, including cloned datasets`
}

func (t *Snapshot) Init() error {
	return t.ConfigOptions.Init()
}

func (t *Snapshot) Configured() error {
	if t.Recursive && !t.Destroy {
		return errors.New("cannot use -R without -d")
	}
	return nil
}

func (t *Snapshot) Snapshot(qsnapshot string, arg ...string) error {
	if !strings.HasPrefix(qsnapshot, "@") {
		return errors.New("snapshot should begin with '@': " + qsnapshot)
	}
	snapshot := qsnapshot[1:]
	return t.ConfigOptions.Run(arg, func(name string, config *Config) error {
		dev := NewDeviceConfigurer(t.Client, config)
		paths, err := dev.FilesystemPaths(name)
		if err != nil {
			return err
		}
		s := &script.Script{Trace: true, DryRun: t.DryRun}
		if t.Destroy {
			if t.Recursive {
				roots := RootPaths(paths)
				for _, path := range roots {
					s.Run("sudo", "zfs", "destroy", "-R", path+"@"+snapshot)
				}
			} else {
				for _, path := range paths {
					s.Run("sudo", "zfs", "destroy", path+"@"+snapshot)
				}
			}
		} else {
			for _, path := range paths {
				s.Run("sudo", "zfs", "snapshot", string(path)+"@"+snapshot)
			}
		}
		return s.Error()
	})
}
