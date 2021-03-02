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
	server, err := t.Client.Server()
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

func (t *LxdOps) Pattern(name string, pattern string) error {
	p := t.Client.NewProperties(name)
	result, err := p.Substitute(pattern)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}
