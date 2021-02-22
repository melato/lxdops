package lxdops

import (
	"melato.org/command"
)

type Trace struct {
	Trace bool `name:"trace,t" usage:"print exec arguments"`
}

type DryRun struct {
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func RootCommand() *command.SimpleCommand {
	var lxdOps LxdOps
	var cmd command.SimpleCommand
	launcher := &Launcher{}
	cmd.Command("launch").Flags(launcher).RunMethodArgs(launcher.Launch).
		Use("<container> <config-file> ...").
		Short("launch a container").
		Example("launch php php.yaml")
	cmd.Command("delete").Flags(launcher).RunMethodArgs(launcher.Delete).Short("delete a stopped container and its profile")
	cmd.Command("rename").Flags(launcher).RunFunc(launcher.Rename).Short("rename a container, filesystems, config file, and rebuild its profile")
	cmd.Command("filesystems").Flags(launcher).RunFunc(launcher.PrintFilesystems).Short("list filesystems")

	configurer := &Configurer{}
	cmd.Command("configure").Flags(configurer).RunMethodArgs(configurer.Run).
		Use("<container> <config-file> ...").
		Short("configure an existing container").
		Example("configure c1 demo.yaml")
	var configOps ConfigOps
	cmd.Command("verify").Flags(&configOps).RunMethodArgs(configOps.Verify).
		Use("<config-file> ...").
		Short("verify config files").
		Example("verify *.yaml")
	cmd.Command("create-devices").Flags(&configOps).RunMethodArgs(configOps.CreateDevices).Use("{container-name} {configfile}...").Short("create devices")
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

	profile.Command("exists").RunMethodArgs(lxdOps.ProfileExists).Use("<profile>").Short("check if a profile exists")
	profile.Command("add-disk").RunMethodArgs(lxdOps.AddDiskDevice).Use("<profile> <source> <path>").Short("add a disk device to a profile")
	cmd.Command("zfsroot").RunMethodE(lxdOps.ZFSRoot).Short("print zfs parent of lxd dataset")
	cmd.Command("current-project").RunMethodE(lxdOps.CurrentProject).Short("print the name of the current project")

	parse := &ParseOp{}
	cmd.Command("parse").Flags(parse).RunMethodArgs(parse.Run).
		Short("parse a config file").
		Use("<config-file>").
		Example("parse test.yaml")

	cmd.Command("description").RunMethodArgs(configOps.Description).
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

	cmd.Long(`lxdops launches or copies containers and creates or clones zfs filesystem devices for them, using yaml config files.  It can also install packages, create users, setup authorized_keys for users, push files, attach profiles, and run scripts.
One of its goals is to facilitate separating the container OS files from the user files, so that the container can be upgraded by relaunching it, thus replacing its OS, rather than upgrading the OS in place.  It is expected that such relaunching can be done by copying a template container and keeping the existing container devices.  The template container can be upgraded with the traditional way, or relaunched from scratch.
A config file provides the recipe for how the container should be created.
Devices are attached to the container via a .devices profile
`)

	return &cmd
}
