package lxdops

import (
	"encoding/json"
	"errors"

	"melato.org/script"
)

type Container struct {
	Name     string   `json:"name"`
	Profiles []string `json:"profiles"`
	State    State    `json:"state"`
}

type State struct {
	Network map[string]*Network `json:"network"`
}

type Network struct {
	Addresses []*Address `json:"addresses"`
}

type Address struct {
	Address string `json:"address"`
	Family  string `json:"family"`
	Netmask string `json:"netmask"`
	Scope   string `json:"scope"`
}

type Project struct {
	Name string `json:name`
}

func ListContainer(name string) (*Container, error) {
	var scr script.Script
	output := scr.Cmd("lxc", "list", name, "--format=json").ToBytes()
	if scr.Error != nil {
		return nil, scr.Error
	}
	var containers []*Container
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
