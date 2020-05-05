package lxdops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"melato.org/export/program"
)

type Ops struct {
	ZFSRootFlag string `name:"zfs-root" usage:"base zfs filesystem" default:"parent of default storage zfs.pool_name"`
	Trace       bool   `usage:"print exec arguments"`
	zfs         program.Program
}

func (t *Ops) Init() error {
	t.Trace = true
	return nil
}

func (t *Ops) Configured() error {
	program.DefaultParams.Trace = t.Trace
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

func (t *Ops) GetPath(dir string) (string, error) {
	if strings.HasPrefix(dir, "/") {
		return dir, nil
	}
	zfsroot, err := t.ZFSRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join("/", zfsroot, dir), nil
}

func (t *Ops) ZFS() program.Program {
	if t.zfs == nil {
		t.zfs = program.NewProgram("zfs").Sudo(true)
	}
	return t.zfs
}

func (t *Ops) GetDefaultDataset() (string, error) {
	lines, err := program.NewProgram("lxc").Lines("storage", "get", "default", "zfs.pool_name")
	if err != nil {
		return "", err
	}
	if len(lines) > 0 {
		fs := lines[0]
		return fs, nil
	}
	return "", errors.New("could not get default zfs.pool_name")
}

func (t *Ops) CloneRepository(rep string) error {
	path, err := t.GetPath("dev")
	if err != nil {
		return err
	}
	dir := filepath.Join(path, rep)
	if !DirExists(dir) {
		fmt.Println("clone repository", rep)
		err = os.MkdirAll(filepath.Dir(dir), 0755)
		if err != nil {
			return err
		}
		git := program.NewProgram("git")
		git.Run("clone", "git:"+rep, dir)
	} else if err != nil {
		return err
	}
	return nil
}

func (t *Ops) waitForNetwork(name string) error {
	for i := 0; i < 30; i++ {
		lines, err := program.NewProgram("lxc").Lines("list", name, "--format=csv", "-c4")
		if err != nil {
			return err
		}
		if len(lines) > 0 {
			ip := lines[0]
			if ip != "" {
				if t.Trace {
					fmt.Println(ip)
				}
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not get ip address for: " + name)
}
