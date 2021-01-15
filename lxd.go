package lxdops

import (
	"encoding/json"
	"errors"

	"github.com/lxc/lxd/shared/api"
	"melato.org/script"
)

func ListContainer(name string) (*api.ContainerFull, error) {
	var scr script.Script
	output := scr.Cmd("lxc", "list", name, "--format=json").ToBytes()
	if scr.Error != nil {
		return nil, scr.Error
	}
	var containers []*api.ContainerFull
	err := json.Unmarshal(output, &containers)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, errors.New("container not found: " + name)
}

func ListContainersForProject(project string) ([]*api.ContainerFull, error) {
	var scr script.Script
	output := scr.Cmd("lxc", "list", "--project", project, "--format=json").ToBytes()
	if scr.Error != nil {
		return nil, scr.Error
	}
	var containers []*api.ContainerFull
	err := json.Unmarshal(output, &containers)
	if err != nil {
		return nil, err
	}
	return containers, err
}
