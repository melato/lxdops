package lxdops

import (
	"errors"
	"os/exec"
	"path/filepath"
	"time"

	"melato.org/script"
)

type Migrate struct {
	PropertyOptions
	FromHost      string
	ConfigFile    string `name:"c" usage:"configFile"`
	FromContainer string
	Container     string
	Snapshot      string `name:"s" usage:"snapshot name"`
	DryRun        bool   `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *Migrate) Init() error {
	t.Snapshot = time.Now().UTC().Format("20060102150405")
	return t.PropertyOptions.Init()
}

func (t *Migrate) Configured() error {
	if len(t.Snapshot) == 0 {
		return errors.New("empty snapshot name")
	}
	if t.FromHost == "" {
		return errors.New("missing -from host")
	}
	if t.ConfigFile == "" {
		return errors.New("missing config file")
	}
	if t.Container == "" {
		return errors.New("missing container")
	}
	if t.FromContainer == "" {
		t.FromContainer = t.Container
	}
	if !filepath.IsAbs(t.ConfigFile) {
		return errors.New("config file should be absolute")
	}
	return t.PropertyOptions.Configured()
}

func (t *Migrate) CopyFilesystems() error {
	var config *Config
	config, err := ReadConfig(t.ConfigFile)
	if err != nil {
		return err
	}
	instance, err := NewInstance(t.GlobalProperties, config, t.Container)
	if err != nil {
		return err
	}
	fromInstance := instance
	if t.FromContainer != t.Container {
		fromInstance, err = NewInstance(t.GlobalProperties, config, t.FromContainer)
		if err != nil {
			return err
		}
	}

	filesystems, err := instance.FilesystemList()
	if err != nil {
		return err
	}
	fromFilesystems, err := fromInstance.Filesystems()
	if err != nil {
		return err
	}
	s := script.Script{Trace: true, DryRun: t.DryRun}
	s.Run("ssh", t.FromHost, "lxdops", "snapshot", "-s", t.Snapshot, "--name", t.FromContainer, t.ConfigFile)
	for _, fs := range filesystems {
		if fs.IsZfs() && !fs.Filesystem.Transient {
			fromFS, ok := fromFilesystems[fs.Id]
			if !ok {
				continue
			}
			send := exec.Command("ssh", t.FromHost, "sudo", "zfs", "send", fromFS.Path+"@"+t.Snapshot)
			receive := exec.Command("sudo", "zfs", "receive", fs.Path)
			s.RunCmd(send, receive)
		}
	}
	return s.Error()
}
