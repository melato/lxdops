package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"melato.org/script"
)

type Ops struct {
	ZFSRootFlag string `name:"zfs-root" usage:"base zfs filesystem" default:"parent of default storage zfs.pool_name"`
	Trace       bool   `name:"trace,t" usage:"print exec arguments"`
}

func (t *Ops) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace}
}

func (t *Ops) Init() error { return nil }

func (t *Ops) Configured() error {
	return nil
}

func (t *Ops) ZFSRoot() (string, error) {
	if t.ZFSRootFlag == "" {
		fs, err := t.GetDefaultDataset()
		if err != nil {
			return "", err
		}
		t.ZFSRootFlag = filepath.Dir(fs)
	}
	return t.ZFSRootFlag, nil
}

func (t *Ops) GetDefaultDataset() (string, error) {
	script := t.NewScript()
	lines := script.Cmd("lxc", "storage", "get", "default", "zfs.pool_name").ToLines()
	if script.Error != nil {
		return "", script.Error
	}
	if len(lines) > 0 {
		fs := lines[0]
		return fs, nil
	}
	return "", errors.New("could not get default zfs.pool_name")
}

func (t *Ops) WaitForNetwork(name string) error {
	for i := 0; i < 30; i++ {
		c, err := ListContainer(name)
		if err != nil {
			return err
		}
		for _, net := range c.State.Network {
			for _, a := range net.Addresses {
				if a.Family == "inet" && a.Scope == "global" {
					fmt.Println(a.Address)
					return nil
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not get ip address for: " + name)
}
