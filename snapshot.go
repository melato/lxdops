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
	return t.ConfigOptions.RunInstances(func(instance *Instance) error {
		filesystems, err := instance.FilesystemList()
		if err != nil {
			return err
		}
		fslist := InstanceFSList(filesystems)
		fslist.Sort()
		s := &script.Script{Trace: true, DryRun: t.DryRun}
		if t.Destroy {
			if t.Recursive {
				roots := fslist.Roots()
				for _, fs := range roots {
					s.Run("sudo", "zfs", "destroy", "-R", fs.Path+"@"+snapshot)
				}
			} else {
				for _, fs := range fslist {
					s.Run("sudo", "zfs", "destroy", fs.Path+"@"+snapshot)
				}
			}
		} else {
			for _, fs := range fslist {
				s.Run("sudo", "zfs", "snapshot", fs.Path+"@"+snapshot)
			}
		}
		return s.Error()
	}, arg...)
}
