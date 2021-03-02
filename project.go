package lxdops

import (
	"errors"
	"fmt"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/util"
)

type ProjectOps struct {
	Client        *LxdClient `name:"-"`
	SourceProject string     `name:"source-project" usage:"project to copy profiles from"`
	TargetProject string     `name:"target-project" usage:"project to copy profiles to "`
}

func (t *ProjectOps) projectServer(server lxd.InstanceServer, project string) (lxd.InstanceServer, error) {
	if project == "" || project == "default" {
		return server, nil
	}
	return server.UseProject(project), nil
}

func (t *ProjectOps) CopyProfiles(profiles []string) error {
	server, err := t.Client.RootServer()
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
	targetProfiles := util.StringSlice(targetProfileNames).ToMap()
	for _, name := range profiles {
		source, _, err := sourceServer.GetProfile(name)
		if err != nil {
			return AnnotateLXDError(t.SourceProject+" "+name, err)
		}
		if !targetProfiles[name] {
			err = targetServer.CreateProfile(api.ProfilesPost{Name: name, ProfilePut: source.ProfilePut})
		} else {
			err = targetServer.UpdateProfile(name, source.ProfilePut, "")

		}
		if err != nil {
			return AnnotateLXDError(t.TargetProject+" "+name, err)
		}
	}
	return nil
}

func (t *ProjectOps) Create(projects ...string) error {
	server, err := t.Client.RootServer()
	if err != nil {
		return err
	}
	projectNames, err := server.GetProjectNames()
	if err != nil {
		return err
	}
	projectSet := util.StringSlice(projectNames).ToMap()
	projectPut := api.ProjectPut{Config: map[string]string{
		"features.images": "false",
	}}
	profile, _, err := server.GetProfile("default")
	if err != nil {
		return errors.New("cannot get default profile")
	}

	for _, project := range projects {
		if !projectSet[project] {
			fmt.Printf("create project %s: %v\n", project, projectPut.Config)
			err := server.CreateProject(api.ProjectsPost{Name: project, ProjectPut: projectPut})
			if err != nil {
				return err
			}
			projectServer := server.UseProject(project)
			fmt.Printf("copy default profile from %s project to %s\n", "default", project)
			err = projectServer.UpdateProfile("default", profile.ProfilePut, "")
			if err != nil {
				return err
			}
		}
	}
	return nil
}
