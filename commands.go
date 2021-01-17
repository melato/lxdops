package lxdops

import (
	"melato.org/command"
)

func RootCommand() *command.SimpleCommand {
	var ops LxdOps
	//ops.Init()
	//ops.Ops = &Ops{}
	//ops.Ops.Init()
	var cmd command.SimpleCommand
	//cmd.Flags(ops.Ops)
	launcher := &Launcher{Ops: ops.Ops}
	cmd.Command("launch").Flags(launcher).RunMethodArgs(launcher.Launch).
		Use("<container> <config-file> ...").
		Short("launch a container").
		Example("launch php php.yaml")
	cmd.Command("delete").Flags(launcher).RunMethodArgs(launcher.Delete)

	configurer := &Configurer{Ops: ops.Ops}
	cmd.Command("configure").Flags(configurer).RunMethodArgs(configurer.Run).
		Use("<container> <config-file> ...").
		Short("configure an existing container").
		Example("configure c1 demo.yaml")
	cmd.Command("verify").Flags(&ops).RunMethodArgs(ops.Verify).
		Use("<config-file> ...").
		Short("verify config files").
		Example("verify *.yaml")
	device := &DeviceCmd{}
	cmd.Command("create-devices").Flags(device).RunMethodArgs(device.Run).Use("{container-name} {configfile}...").Short("create devices")
	/* add devices:
	lxdops device add -p a.host -d /z/host/a -s 1 {configfile}...
	- create subdirectories
	- change ownership
	- add devices to profile, with optional suffix
	*/
	profile := cmd.Command("profile").Short("profile utilities")
	profileConfigurer := &ProfileConfigurer{}
	profile.Command("list").Flags(profileConfigurer).RunMethodArgs(profileConfigurer.List).Short("list config profiles")
	profile.Command("diff").Flags(profileConfigurer).RunMethodArgs(profileConfigurer.Diff).Short("compare container profiles with config")
	profile.Command("apply").Flags(profileConfigurer).RunMethodArgs(profileConfigurer.Apply).Short("apply the config profiles to a container")
	profile.Command("reorder").Flags(profileConfigurer).RunMethodArgs(profileConfigurer.Reorder).Short("reorder container profiles to match config order")

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
	containerCmd.Command("network").RunMethodArgs(containerOps.Network)
	containerCmd.Command("wait").RunMethodArgs(containerOps.Wait)

	return &cmd
}
