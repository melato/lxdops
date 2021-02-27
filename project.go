package lxdops

import (
	"errors"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

type ProjectOps struct {
	Client        *LxdClient `name:"-"`
	SourceProject string     `name:"from" usage:"source project"`
	TargetProject string     `name:"to" usage:"target project"`
}

func (t *ProjectOps) projectServer(server lxd.InstanceServer, project string) (lxd.InstanceServer, error) {
	if project == "" || project == "default" {
		return server, nil
	}
	sp := server.UseProject(project)
	if sp == nil {
		return nil, errors.New("no such project: " + project)
	}
	return sp, nil
}

func (t *ProjectOps) CopyProfiles(profiles []string) error {
	server, err := t.Client.Server()
	if err != nil {
		return err
	}
	sourceServer, err := t.projectServer(server, t.SourceProject)
	if err != nil {
		return err
	}
	targetServer, err := t.projectServer(server, t.TargetProject)
	if err != nil {
		return err
	}
	targetProfileNames, err := targetServer.GetProfileNames()
	if err != nil {
		return err
	}
	targetProfiles := make(map[string]bool)
	for _, name := range targetProfileNames {
		targetProfiles[name] = true
	}
	for _, name := range profiles {
		source, _, err := sourceServer.GetProfile(name)
		if err != nil {
			return err
		}
		if !targetProfiles[name] {
			err = targetServer.CreateProfile(api.ProfilesPost{Name: name, ProfilePut: source.ProfilePut})
		} else {
			err = targetServer.UpdateProfile(name, source.ProfilePut, "")

		}
		if err != nil {
			return err
		}
	}
	return nil
}
