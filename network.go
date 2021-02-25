package lxdops

import (
	"encoding/csv"
	"errors"
	"os"
	"strings"

	lxd "github.com/lxc/lxd/client"
)

type HostAddress struct {
	Name    string
	Address string
}

type NetworkManager struct {
	Client *LxdClient
}

func (t *NetworkManager) ParseAddress(addr string) string {
	i := strings.Index(addr, " ")
	if i > 0 {
		return addr[0:i]
	}
	return ""
}

func (t *NetworkManager) GetProjectAddresses(server lxd.ContainerServer, project string) ([]*HostAddress, error) {
	containers, err := server.GetContainersFull()
	if err != nil {
		return nil, err
	}
	var addresses []*HostAddress

	for _, c := range containers {
		if c.State == nil || c.State.Network == nil {
			continue
		}
		for _, net := range c.State.Network {
			for _, a := range net.Addresses {
				if a.Family == "inet" && a.Scope == "global" {
					addresses = append(addresses, &HostAddress{Name: QualifiedContainerName(project, c.Name), Address: a.Address})
				}
			}
		}
	}
	return addresses, nil
}

func (t *NetworkManager) GetAddresses() ([]*HostAddress, error) {
	server, err := t.Client.Server()
	if err != nil {
		return nil, err
	}
	projects, err := server.GetProjects()
	if err != nil {
		return nil, err
	}
	var addresses []*HostAddress
	for _, project := range projects {
		paddresses, err := t.GetProjectAddresses(server, project.Name)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, paddresses...)
	}
	return addresses, nil
}

func (t *NetworkManager) WriteAddresses(addresses []*HostAddress, file string, headers bool) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	var csv = csv.NewWriter(f)
	if headers {
		csv.Write([]string{"address", "name"})
	}
	for _, a := range addresses {
		csv.Write([]string{a.Address, a.Name})
	}
	csv.Flush()
	return csv.Error()
}

type NetworkOp struct {
	Client     *LxdClient `name:"-"`
	OutputFile string     `name:"o" usage:"output file"`
	Headers    bool       `name:"headers" usage:"include headers"`
}

func (t *NetworkOp) ExportAddresses() error {
	if t.OutputFile == "" {
		return errors.New("Missing output file")
	}
	net := &NetworkManager{Client: t.Client}
	containers, err := net.GetAddresses()

	if err != nil {
		return err
	}

	return net.WriteAddresses(containers, t.OutputFile, t.Headers)
}
