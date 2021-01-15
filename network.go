package lxdops

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"melato.org/lxdops/lxd"
	"melato.org/script"
)

type HostAddress struct {
	Name    string
	Address string
}

type NetworkManager struct {
}

func (t *NetworkManager) ParseAddress(addr string) string {
	i := strings.Index(addr, " ")
	if i > 0 {
		return addr[0:i]
	}
	return ""
}

func (t *NetworkManager) GetProjectAddresses(project string) ([]*HostAddress, error) {
	var s script.Script
	// lxc list -c ns4 --format=csv
	output := s.Cmd("lxc", "list", "--project", project, "-c", "n4", "--format=csv").ToBytes()
	if s.Error != nil {
		return nil, s.Error
	}
	var addresses []*HostAddress
	r := csv.NewReader(bytes.NewReader(output))
	for {
		fields, _ := r.Read()
		if fields == nil {
			break
		}
		if fields[1] != "" {
			address := t.ParseAddress((fields[1]))
			if address != "" {
				addresses = append(addresses, &HostAddress{Name: fields[0], Address: address})
			}
		}
	}
	return addresses, nil
}

func (t *NetworkManager) GetProjects() ([]string, error) {
	var s script.Script
	output := s.Cmd("lxc", "project", "list", "--format=json").ToBytes()
	if s.Error != nil {
		return nil, s.Error
	}
	var projects []*lxd.Project
	err := json.Unmarshal(output, &projects)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names, nil
}

func (t *NetworkManager) GetAddresses() ([]*HostAddress, error) {
	projects, err := t.GetProjects()
	if err != nil {
		return nil, err
	}
	var addresses []*HostAddress
	for _, project := range projects {
		paddresses, err := t.GetProjectAddresses(project)
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
	OutputFile string `name:"o" usage:"output file"`
	Headers    bool   `name:"headers" usage:"include headers"`
}

func (t *NetworkOp) ExportAddresses() error {
	if t.OutputFile == "" {
		return errors.New("Missing output file")
	}
	var net NetworkManager
	containers, err := net.GetAddresses()

	if err != nil {
		return err
	}

	return net.WriteAddresses(containers, t.OutputFile, t.Headers)
}
