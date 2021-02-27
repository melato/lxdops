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
		return dev.IterateFilesystems(name, func(path string) error {
			s := &script.Script{Trace: true, DryRun: t.DryRun}
			if t.Destroy {
				args := []string{"zfs", "destroy", path + "@" + snapshot}
				if t.Recursive {
					args = append(args, "-R")
				}
				s.Run("sudo", args...)
			} else {
				s.Run("sudo", "zfs", "snapshot", path+"@"+snapshot)
			}
			return s.Error()
		})
	})
}
