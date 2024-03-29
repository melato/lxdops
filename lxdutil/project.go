package lxdutil

import (
	"errors"
	"fmt"

	"github.com/canonical/lxd/shared/api"
	"melato.org/lxdops/util"
)

type ProjectCopyProfiles struct {
	Client        *LxdClient `name:"-"`
	SourceProject string     `name:"source-project" usage:"project to copy profiles from"`
	TargetProject string     `name:"target-project" usage:"project to copy profiles to"`
}

type ProjectCreate struct {
	Client *LxdClient `name:"-"`
}

func (t *ProjectCopyProfiles) CopyProfiles(profiles []string) error {
	sourceServer, err := t.Client.ProjectServer(t.SourceProject)
	if err != nil {
		return err
	}
	targetServer, err := t.Client.ProjectServer(t.TargetProject)
	if err != nil {
		return err
	}
	targetProfileNames, err := targetServer.GetProfileNames()
	if err != nil {
		return err
	}
	targetProfiles := util.StringSlice(targetProfileNames).ToSet()
	for _, name := range profiles {
		source, _, err := sourceServer.GetProfile(name)
		if err != nil {
			return AnnotateLXDError(t.SourceProject+" "+name, err)
		}
		if !targetProfiles.Contains(name) {
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

func (t *ProjectCreate) Create(projects ...string) error {
	server, err := t.Client.RootServer()
	if err != nil {
		return err
	}
	projectNames, err := server.GetProjectNames()
	if err != nil {
		return err
	}
	projectSet := util.StringSlice(projectNames).ToSet()
	projectPut := api.ProjectPut{Config: map[string]string{
		"features.images": "false",
	}}
	projectPut.Config["features.profiles"] = "true"
	profile, _, err := server.GetProfile("default")
	if err != nil {
		return errors.New("cannot get default profile")
	}

	for _, project := range projects {
		if !projectSet.Contains(project) {
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
