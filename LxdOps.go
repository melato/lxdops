package lxdops

import (
	"fmt"
	"path/filepath"
)

type LxdOps struct {
	Client *LxdClient `name:"-"`
	Trace  bool       `name:"trace,t" usage:"print exec arguments"`
}

func (t *LxdOps) ZFSRoot() error {
	dataset, err := t.Client.GetDefaultDataset()
	if err != nil {
		return err
	}
	fmt.Println(filepath.Dir(dataset))
	return nil
}

func (t *LxdOps) AddDiskDevice(profile, source, path string) error {
	project := t.Client.CurrentProject()
	server, err := t.Client.ProjectServer(project)
	if err != nil {
		return err
	}
	p, _, err := server.GetProfile(profile)
	if err != nil {
		return AnnotateLXDError(profile, err)
	}

	device := RandomDeviceName()
	p.Devices[device] = map[string]string{"type": "disk", "path": path, "source": source}
	return server.UpdateProfile(profile, p.ProfilePut, "")
}

func (t *LxdOps) ProfileExists(profile string) error {
	project := t.Client.CurrentProject()
	server, err := t.Client.ProjectServer(project)
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
