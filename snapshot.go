package lxdops

import (
	"errors"
	"time"
)

type SnapshotParams struct {
	DryRun    bool   `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Snapshot  string `name:"s" usage:"short snapshot name"`
	Destroy   bool   `name:"d" usage:"destroy snapshots"`
	Recursive bool   `name:"R" usage:"zfs destroy -R: Recursively destroy all dependents, including cloned datasets`
}

type Snapshot struct {
	ConfigOptions
	SnapshotParams
}

func (t *Snapshot) Init() error {
	t.SnapshotParams.Snapshot = "snap" + time.Now().UTC().Format("20060102150405")
	return t.ConfigOptions.Init()
}

func (t *Snapshot) Configured() error {
	if len(t.SnapshotParams.Snapshot) == 0 {
		return errors.New("empty snapshot name")
	}
	if t.Recursive && !t.Destroy {
		return errors.New("cannot use -R without -d")
	}
	return nil
}

func (t *Snapshot) Snapshot(instance *Instance) error {
	return instance.Snapshot(t.SnapshotParams)
}
