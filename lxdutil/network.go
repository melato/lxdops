package lxdutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
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

func (t *NetworkManager) GetProjectAddresses(server lxd.InstanceServer, instanceType api.InstanceType, project string, family string) ([]*HostAddress, error) {
	containers, err := server.UseProject(project).GetInstancesFull(instanceType)
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
		for _, itype := range []api.InstanceType{api.InstanceTypeContainer, api.InstanceTypeVM} {
			paddresses, err := t.GetProjectAddresses(server, itype, project.Name, family)
			if err != nil {
				return nil, err
			}
			addresses = append(addresses, paddresses...)
		}
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
	Format     string     `name:"format" usage:"include format: csv | yaml"`
	Headers    bool       `name:"headers" usage:"include headers"`
	Family     string     `name:"family" usage:"network family: inet | inet6"`
}

func (t *NetworkOp) Init() error {
	t.Family = "inet"
	t.Format = "csv"
	return t.Client.Init()
}

func (t *NetworkOp) ExportAddresses() error {
	net := &NetworkManager{Client: t.Client}
	containers, err := net.GetAddresses(t.Family)

	if err != nil {
		return err
	}

	var printer AddressPrinter
	switch t.Format {
	case "csv":
		printer = &CsvAddressPrinter{Headers: t.Headers}
	case "yaml":
		printer = &YamlAddressPrinter{}
	default:
		return fmt.Errorf("unrecognized format: %s", t.Format)
	}

	var out io.WriteCloser
	if t.OutputFile == "" {
		out = os.Stdout
	} else {
		out, err = os.Create(t.OutputFile)
		if err != nil {
			return err
		}
		defer out.Close()
	}
	return printer.Print(containers, out)
}
