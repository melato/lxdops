package lxdops

import (
	"errors"

	"melato.org/command"
	"melato.org/shorewall"
	shorewall_commands "melato.org/shorewall/commands"
)

type ShorewallRulesOp struct {
	RulesFile    string `name:"f,rules-file" usage:"rules file"`
	FirstSshPort int    `name:"ssh,p" usage:"first ssh port"`
}

func (t *ShorewallRulesOp) Init() error {
	t.FirstSshPort = 6002
	return nil
}

func (t *ShorewallRulesOp) Run() error {
	if t.RulesFile == "" {
		return errors.New("Missing rules file")
	}
	var net NetworkManager
	containers, err := net.GetAddresses()

	if err != nil {
		return err
	}

	addresses := make([]*shorewall.HostAddress, len(containers))
	for i, c := range containers {
		addresses[i] = &shorewall.HostAddress{Name: c.Name, Address: c.Address}
	}
	sw := shorewall.Shorewall{FirstSshPort: t.FirstSshPort}
	return sw.GenerateRules(addresses, t.RulesFile)
}

func AddShorewallCommands(parent *command.SimpleCommand) {
	cmd := parent.Command("shorewall")
	var interfacesCmd shorewall_commands.InterfacesCmd
	cmd.Command("interfaces").Flags(&interfacesCmd).RunMethodE(interfacesCmd.Run)
	var rulesOp ShorewallRulesOp
	cmd.Command("rules").Flags(&rulesOp).RunMethodE(rulesOp.Run).Short("generate shorewall rules")
}
