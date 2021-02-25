package lxdops

import (
	"errors"
	"fmt"

	"melato.org/script/v2"
)

type LxdOps struct {
	Client *LxdClient `name:"-"`
	Trace  bool       `name:"trace,t" usage:"print exec arguments"`
}

func (t *LxdOps) ZFSRoot() error {
	path, err := ZFSRoot()
	if err == nil {
		fmt.Println(path)
	}
	return err
}

func (t *LxdOps) AddDiskDevice(args []string) error {
	if len(args) != 3 {
		return errors.New("Usage: <profile> <source> <path>")
	}
	profile := args[0]
	source := args[1]
	path := args[2]
	device := RandomDeviceName()
	script := &script.Script{Trace: t.Trace}
	script.Run("lxc", "profile", "device", "add", profile, device, "disk", "path="+path, "source="+source)
	return script.Error()
}

func (t *LxdOps) ProfileExists(profile string) error {
	server, err := t.Client.Server()
	if err != nil {
		return err
	}
	prof, _, err := server.GetProfile(profile)
	if err != nil {
		return err
	}
	fmt.Println(prof.Name)
	return nil
}

func (t *LxdOps) CurrentProject() error {
	project, err := CurrentProject()
	if err != nil {
		return err
	}
	fmt.Println(project)
	return nil
}
