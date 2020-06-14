package lxdops

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"

	"melato.org/export/command"
	"melato.org/export/program"
	"melato.org/shorewall"
)

type ShorewallRulesOp struct {
	command.Base
	RulesFile    string `name:"f,rules-file" usage:"rules file"`
	FirstSshPort int    `name:"ssh,p" usage:"first ssh port"`
}

func (t *ShorewallRulesOp) Init() error {
	t.FirstSshPort = 6002
	return nil
}

func ParseAddress(addr string) string {
	i := strings.Index(addr, " ")
	if i > 0 {
		return addr[0:i]
	}
	return ""
}

func (t *ShorewallRulesOp) GetProjectAddresses(project string) ([]*shorewall.HostAddress, error) {
	// lxc list -c ns4 --format=csv
	cmd, err := program.NewProgram("lxc").Cmd("list", "--project", project, "-c", "n4", "--format=csv")
	if err != nil {
		return nil, err
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var addresses []*shorewall.HostAddress
	r := csv.NewReader(bytes.NewReader(output))
	for {
		fields, _ := r.Read()
		if fields == nil {
			break
		}
		if fields[1] != "" {
			address := ParseAddress((fields[1]))
			if address != "" {
				addresses = append(addresses, &shorewall.HostAddress{Name: fields[0], Address: address})
			}
		}
	}
	return addresses, nil
}

func (t *ShorewallRulesOp) GetProjects() ([]string, error) {
	cmd, err := program.NewProgram("lxc").Cmd("project", "list", "--format=json")
	if err != nil {
		return nil, err
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var projects []*Project
	err = json.Unmarshal(output, &projects)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names, nil
}

func (t *ShorewallRulesOp) GetAddresses() ([]*shorewall.HostAddress, error) {
	projects, err := t.GetProjects()
	if err != nil {
		return nil, err
	}
	var addresses []*shorewall.HostAddress
	for _, project := range projects {
		paddresses, err := t.GetProjectAddresses(project)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, paddresses...)
	}
	return addresses, nil
}

func (t *ShorewallRulesOp) Run(args []string) error {
	if t.RulesFile == "" {
		return errors.New("Missing rules file")
	}
	containers, err := t.GetAddresses()

	if err != nil {
		return err
	}

	sw := shorewall.Shorewall{FirstSshPort: t.FirstSshPort}
	return sw.GenerateRules(containers, t.RulesFile)
}

func (op *ShorewallRulesOp) Usage() *command.Usage {
	return &command.Usage{
		Short: "generate shorewall rules",
	}
}
