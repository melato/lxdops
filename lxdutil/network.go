package lxdutil

import (
	"encoding/csv"
	"io"
	"os"
	"strings"

	lxd "github.com/lxc/lxd/client"
)

func QualifiedContainerName(project string, container string) string {
	if project == DefaultProject {
		return container
	}
	return project + "_" + container
}

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

func (t *NetworkManager) GetProjectAddresses(server lxd.ContainerServer, project string, family string) ([]*HostAddress, error) {
	containers, err := server.UseProject(project).GetContainersFull()
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
				if a.Family == family && a.Scope == "global" {
					addresses = append(addresses, &HostAddress{Name: QualifiedContainerName(project, c.Name), Address: a.Address})
				}
			}
		}
	}
	return addresses, nil
}

func (t *NetworkManager) GetAddresses(family string) ([]*HostAddress, error) {
	server, err := t.Client.RootServer()
	if err != nil {
		return nil, err
	}
	projects, err := server.GetProjects()
	if err != nil {
		return nil, err
	}
	var addresses []*HostAddress
	for _, project := range projects {
		paddresses, err := t.GetProjectAddresses(server, project.Name, family)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, paddresses...)
	}
	return addresses, nil
}

func (t *NetworkManager) printAddresses(addresses []*HostAddress, headers bool, writer io.Writer) error {
	var csv = csv.NewWriter(writer)
	if headers {
		csv.Write([]string{"address", "name"})
	}
	for _, a := range addresses {
		csv.Write([]string{a.Address, a.Name})
	}
	csv.Flush()
	return csv.Error()
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
	Family     string     `name:"family" usage:"network family: inet | inet6"`
}

func (t *NetworkOp) Init() error {
	t.Family = "inet"
	return t.Client.Init()
}

func (t *NetworkOp) ExportAddresses() error {
	net := &NetworkManager{Client: t.Client}
	containers, err := net.GetAddresses(t.Family)

	if err != nil {
		return err
	}

	if t.OutputFile == "" {
		return net.printAddresses(containers, t.Headers, os.Stdout)
	} else {
		f, err := os.Create(t.OutputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		return net.printAddresses(containers, t.Headers, f)
	}
}
