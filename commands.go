package lxdops

import (
	"fmt"

	"melato.org/export/command"
	shorewall_commands "melato.org/shorewall/commands"
)

type RootCommand struct {
	command.Base
	Ops *Ops
}

func (t *RootCommand) Init() error {
	t.Ops = &Ops{}
	return t.Ops.Init()
}

func (t *RootCommand) Configured() error {
	return t.Ops.Configured()
}

func (t *RootCommand) Usage() *command.Usage {
	return &command.Usage{
		Short:   "launch and configure containers using xml configuration files",
		Example: "launch -c php/php -d z/template/drupal7@x",
	}
}

func (t *RootCommand) Commands() map[string]command.Command {
	commands := make(map[string]command.Command)
	commands["launch"] = &LaunchOp{Launcher: &Launcher{Ops: t.Ops}}
	commands["configure"] = &ConfigureOp{Configurer: NewConfigurer(t.Ops)}
	commands["verify"] = &VerifyOp{}
	commands["parse"] = &ParseOp{}
	commands["shorewall"] = &ShorewallCommand{}
	commands["zfsroot"] = (&command.SimpleCommand{}).RunMethodArgs(t.ZFSRoot)
	device := &DeviceConfigurer{Ops: t.Ops}
	commands["create-devices"] = (&command.SimpleCommand{}).Flags(device).RunMethodArgs(device.Run).Use("{name} {configfile}...").Short("create devices")
	commands["version"] = (&command.SimpleCommand{}).RunMethod(t.Version)
	return commands
}

func (t *RootCommand) ZFSRoot(args []string) error {
	path, err := t.Ops.ZFSRoot()
	if err == nil {
		fmt.Println(path)
	}
	return err
}

func (t *RootCommand) Version() {
	fmt.Println(Version)
}

type ShorewallCommand struct {
	command.Base
}

func (t *ShorewallCommand) Commands() map[string]command.Command {
	commands := make(map[string]command.Command)
	commands["interfaces"] = &shorewall_commands.InterfacesCmd{}
	commands["rules"] = &ShorewallRulesOp{}
	return commands
}
