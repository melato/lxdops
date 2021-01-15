package lxdops

import (
	"encoding/json"
	"errors"

	lxd "github.com/lxc/lxd/shared/api"
	//"melato.org/lxdops/lxd"
	"melato.org/script"
)

func ListContainer(name string) (*lxd.ContainerFull, error) {
	var scr script.Script
	output := scr.Cmd("lxc", "list", name, "--format=json").ToBytes()
	if scr.Error != nil {
		return nil, scr.Error
	}
	var containers []*lxd.ContainerFull
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

func ListContainersForProject(project string) ([]*lxd.ContainerFull, error) {
	var scr script.Script
	output := scr.Cmd("lxc", "list", "--project", project, "--format=json").ToBytes()
	if scr.Error != nil {
		return nil, scr.Error
	}
	var containers []*lxd.ContainerFull
	err := json.Unmarshal(output, &containers)
	if err != nil {
		return nil, err
	}
	return containers, err
}
