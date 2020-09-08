package lxdops

import (
	"fmt"
	"errors"

	"melato.org/export/command"
	shorewall_commands "melato.org/shorewall/commands"
)

type LxdOps struct {
	Ops *Ops
}

func (t *LxdOps) Init() error {
	t.Ops = &Ops{}
	return t.Ops.Init()
}

func (t *LxdOps) Configured() error {
	return t.Ops.Configured()
}

func (t *LxdOps) Usage() *command.Usage {
	return &command.Usage{
		Short:   "launch and configure containers using xml configuration files",
		Example: "launch -c php/php -d z/template/drupal7@x",
	}
}

func RootCommand() command.Command {
	var ops LxdOps
	ops.Ops = &Ops{}
	ops.Ops.Init()
	var cmd command.SimpleCommand
	launcher := &Launcher{Ops: ops.Ops}
	cmd.Command("launch").Flags(launcher).RunMethodArgs(launcher.Run).
		Use("<container> <config-file> ...").
		Short("launch a container").
		Example("launch php php.yaml")

	configurer := NewConfigurer(ops.Ops)
	cmd.Command("configure").Flags(configurer).RunMethodArgs(configurer.Run).
		Use("<container> <config-file> ...").
		Short("configure an existing container").
		Example("configure c1 demo.yaml")
	
	cmd.Command("verify").RunMethodArgs(ops.Verify).
		Use("<config-file> ...").
		Short("verify config files").
		Example("verify *.yaml")
	cmd.Command("version").RunMethod(ops.Version)
	device := &DeviceConfigurer{Ops: ops.Ops}
	cmd.Command("create-devices").Flags(device).RunMethodArgs(device.Run).Use("{name} {configfile}...").Short("create devices")
	cmd.Command("zfsroot").RunMethodArgs(ops.ZFSRoot)

	parse := &ParseOp{}
	cmd.Command("parse").Flags(parse).RunMethodArgs(parse.Run).
		Short("parse a config file").
		Use("<config-file>").
		Example("parse test.yaml")	

	cmd.Command("description").RunMethodArgs(ops.Description).
		Short("print the description of a config file").
		Use("<config-file>").
		Example("test.yaml")	

	shorewallCmd := cmd.Command("shorewall")
	
	var interfacesCmd shorewall_commands.InterfacesCmd
	shorewallCmd.Command("interfaces").Flags(&interfacesCmd).RunMethodE(interfacesCmd.Run)
	var rulesOp ShorewallRulesOp
	shorewallCmd.Command("rules").Flags(&rulesOp).RunMethodE(rulesOp.Run).Short("generate shorewall rules")
	return &cmd
}

func (t *LxdOps) Verify(args []string) error {
	for _, configFile := range args {
		var err error
		var config *Config
		config, err = ReadConfig(configFile)
		isValid := false
		if err != nil {
			fmt.Println(err)
		}
		if err == nil {
			isValid = config.Verify()
		}
		fmt.Println(configFile, isValid)
	}
	return nil
}

func (t *LxdOps) ZFSRoot(args []string) error {
	path, err := t.Ops.ZFSRoot()
	if err == nil {
		fmt.Println(path)
	}
	return err
}

func (t *LxdOps) Version() {
	fmt.Println(Version)
}


/** Print the description of a config file. */
func (t *LxdOps) Description(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: <config.yaml>")
	}
	var err error
	var config *Config
	config, err = ReadConfig(args[0])
	if err != nil {
		return err
	}
	fmt.Println(config.Description)
	return nil
}

