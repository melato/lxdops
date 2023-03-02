package lxdutil

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"melato.org/yaml"
)

type ProfileOps struct {
	Client *LxdClient `name:"-"`
	Dir    string     `name:"d" usage:"export directory"`
}

func (t *ProfileOps) ExportProfile(server lxd.InstanceServer, name string) error {
	profile, _, err := server.GetProfile(name)
	if err != nil {
		return fmt.Errorf("%w: %s", err, name)
	}

	data, err := yaml.Marshal(&profile.ProfilePut)
	if err != nil {
		return err
	}

	file := path.Join(t.Dir, name)
	return os.WriteFile(file, []byte(data), 0644)
}

func (t *ProfileOps) Export(profiles ...string) error {
	server, err := t.Client.CurrentServer()
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		err = t.ExportProfile(server, profile)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *ProfileOps) ImportProfile(server lxd.InstanceServer, file string, existingProfiles map[string]bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var profile api.ProfilePut
	err = yaml.Unmarshal(data, &profile)
	if err != nil {
		return err
	}
	name := filepath.Base(file)
	_, exists := existingProfiles[name]
	if exists {
		return server.UpdateProfile(name, profile, "")
	} else {
		var post api.ProfilesPost
		post.ProfilePut = profile
		post.Name = name
		return server.CreateProfile(post)
	}
}

func (t *ProfileOps) Import(files []string) error {
	server, err := t.Client.CurrentServer()
	if err != nil {
		return err
	}
	profiles := make(map[string]bool)
	names, err := server.GetProfileNames()
	if err != nil {
		return err
	}
	for _, name := range names {
		profiles[name] = true
	}

	for _, file := range files {
		err := t.ImportProfile(server, file, profiles)
		if err != nil {
			return err
		}
	}
	return nil
}
