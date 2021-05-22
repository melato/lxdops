package lxdops

import (
	"fmt"
	"path/filepath"

	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/lxdutil"
)

type AddDisk struct {
	Client   *lxdutil.LxdClient `name:"-"`
	Readonly bool               `name:"r" usage:"readonly"`
	Create   bool               `name:"c" usage:"create profile, if it does not exist"`
}

func CreateDeviceName(path string, exists func(string) bool) string {
	var name string
	sep := string(filepath.Separator)
	for {
		base := filepath.Base(path)
		if base == sep || base == "." {
			break
		}
		if name == "" {
			name = base
		} else {
			name = base + "_" + name
		}
		if !exists(name) {
			return name
		}
		path = filepath.Dir(path)
	}
	return RandomDeviceName()
}

func (t *AddDisk) Add(profile, source, path string) error {
	project := t.Client.CurrentProject()
	server, err := t.Client.ProjectServer(project)
	if err != nil {
		return err
	}
	exists := lxdutil.InstanceServer{server}.ProfileExists(profile)
	if !exists && !t.Create {
		return fmt.Errorf("profile %s does not exist", profile)
	}
	deviceMap := map[string]string{"type": "disk", "path": path, "source": source}
	if t.Readonly {
		deviceMap["readonly"] = "true"
	}
	if exists {
		p, _, err := server.GetProfile(profile)
		if err != nil {
			return lxdutil.AnnotateLXDError(profile, err)
		}
		deviceName := CreateDeviceName(path, func(name string) bool { _, found := p.Devices[name]; return found })
		p.Devices[deviceName] = deviceMap
		return server.UpdateProfile(profile, p.ProfilePut, "")
	} else {
		deviceName := CreateDeviceName(path, func(string) bool { return false })
		post := api.ProfilesPost{Name: profile, ProfilePut: api.ProfilePut{
			Devices: map[string]map[string]string{deviceName: deviceMap},
		}}
		return server.CreateProfile(post)
	}
}
