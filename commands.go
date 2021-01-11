package lxdops

import (
	"errors"
	"fmt"
	"io"
	"os"

	"melato.org/command"
	"melato.org/script"
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

func RootCommand() *command.SimpleCommand {
	var ops LxdOps
	ops.Ops = &Ops{}
	ops.Ops.Init()
	var cmd command.SimpleCommand
	cmd.Flags(ops.Ops)
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
	cmd.Command("version").RunMethod(ops.Version).Short("print program version")
	device := &DeviceCmd{Ops: ops.Ops}
	cmd.Command("create-devices").Flags(device).RunMethodArgs(device.Run).Use("{container-name} {configfile}...").Short("create devices")
	/* add devices:
	lxdops device add -p a.host -d /z/host/a -s 1 {configfile}...
	- create subdirectories
	- change ownership
	- add devices to profile, with optional suffix
	*/
	profile := cmd.Command("profile").Short("profile utilities")
	profileConfigurer := NewProfileConfigurer(ops.Ops)
	profile.Command("apply").Flags(profileConfigurer).RunMethodArgs(profileConfigurer.Apply)
	profile.Command("list").Flags(profileConfigurer).RunMethodArgs(profileConfigurer.List)
	profile.Command("diff").Flags(profileConfigurer).RunMethodArgs(profileConfigurer.Diff)

	profile.Command("exists").RunMethodArgs(ops.ProfileExists).Use("<profile>").Short("check if a profile exists")
	profile.Command("add-disk").RunMethodArgs(ops.AddDiskDevice).Use("<profile> <source> <path>").Short("add a disk device to a profile")
	cmd.Command("zfsroot").RunMethodArgs(ops.ZFSRoot).Short("print zfs parent of lxd dataset")

	parse := &ParseOp{}
	cmd.Command("parse").Flags(parse).RunMethodArgs(parse.Run).
		Short("parse a config file").
		Use("<config-file>").
		Example("parse test.yaml")

	cmd.Command("description").RunMethodArgs(ops.Description).
		Short("print the description of a config file").
		Use("<config-file>").
		Example("test.yaml")

	var networkOp NetworkOp
	cmd.Command("addresses").Flags(&networkOp).RunMethodE(networkOp.ExportAddresses).Short("export container addresses")

	var containerOps ContainerOps
	containerCmd := cmd.Command("container")
	containerCmd.Command("profiles").RunMethodArgs(containerOps.Profiles)

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

func (t *LxdOps) AddDiskDevice(args []string) error {
	if len(args) != 3 {
		return errors.New("Usage: <profile> <source> <path>")
	}
	profile := args[0]
	source := args[1]
	path := args[2]
	device := RandomDeviceName()
	script := &script.Script{Trace: t.Ops.Trace}
	script.Run("lxc", "profile", "device", "add", profile, device, "disk", "path="+path, "source="+source)
	return script.Error
}

type NullWriter struct{ io.Writer }

func (t *NullWriter) Write(p []byte) (n int, err error) { return len(p), nil }

func (t *LxdOps) ProfileExists(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: <profile>")
	}
	profile := args[0]
	script := &script.Script{Trace: t.Ops.Trace}
	cmd := script.Cmd("lxc", "profile", "get", profile, "x")
	cmd.Cmd.Stdout = &NullWriter{}
	cmd.MergeStderr()
	cmd.Run()
	if script.Error == nil {
		if t.Ops.Trace {
			fmt.Printf("profile %s exists\n", profile)
		}
		os.Exit(0)
	} else {
		if t.Ops.Trace {
			fmt.Printf("profile %s does not exist\n", profile)
		}
		os.Exit(1)
	}
	return script.Error
}
