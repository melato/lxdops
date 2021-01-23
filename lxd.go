package lxdops

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

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

func WaitForNetwork(name string) error {
	for i := 0; i < 30; i++ {
		c, err := ListContainer(name)
		if err != nil {
			return err
		}
		if c.State != nil {
			for _, net := range c.State.Network {
				for _, a := range net.Addresses {
					if a.Family == "inet" && a.Scope == "global" {
						fmt.Println(a.Address)
						return nil
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not get ip address for: " + name)
}

func GetDefaultDataset() (string, error) {
	script := script.Script{}
	lines := script.Cmd("lxc", "storage", "get", "default", "zfs.pool_name").ToLines()
	if script.Error != nil {
		return "", script.Error
	}
	if len(lines) > 0 {
		fs := lines[0]
		return fs, nil
	}
	return "", errors.New("could not get default zfs.pool_name")
}

func ZFSRoot() (string, error) {
	fs, err := GetDefaultDataset()
	if err != nil {
		return "", err
	}
	zfsroot := filepath.Dir(fs)
	return zfsroot, nil
}

func ProfileExists(profile string) bool {
	// Not sure what profile get does, but it returns an error if the profile doesn't exist.
	// "x" is a key.  It doesn't matter what key we use for our purpose.
	script := script.Script{}
	return script.Cmd("lxc", "profile", "get", profile, "x").MergeStderr().ToNull()
}
