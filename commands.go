package lxdops

import (
	_ "embed"

	"melato.org/command"
	"melato.org/lxdops/usage"
)

//go:embed commands.yaml
var usageData []byte

type Trace struct {
	Trace bool `name:"trace,t" usage:"print exec arguments"`
}

type DryRun struct {
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func RootCommand() *command.SimpleCommand {
	client := &LxdClient{}
	var cmd command.SimpleCommand
	cmd.Flags(client)
	launcher := &Launcher{Client: client}
	cmd.Command("launch").Flags(launcher).RunFunc(launcher.Launch).
		Use("<container> <config-file> ...").
		Short("launch a container").
		Example("launch php php.yaml")
	cmd.Command("delete").Flags(launcher).RunFunc(launcher.Delete).Short("delete a stopped container and its profile")
	cmd.Command("rebuild").Flags(launcher).RunFunc(launcher.Rebuild).Short("stop, delete, launch").
		Long("Rebuild stops, deletes, and relaunches the container.")
	cmd.Command("rename").Flags(launcher).RunFunc(launcher.Rename).Short("rename an instance").
		Long("Renames the container, its filesystems, and its devices profile")
	cmd.Command("filesystems").Flags(launcher).RunFunc(launcher.PrintFilesystems).Short("list filesystems")
	cmd.Command("devices").Flags(launcher).RunFunc(launcher.PrintDevices).Short("list devices")

	snapshot := &Snapshot{Client: client}
	cmd.Command("snapshot").Flags(snapshot).RunFunc(snapshot.Snapshot).Short("snapshot filesystems").Use("@{snapshot} {config-file}...")

	configurer := &Configurer{Client: client}
	cmd.Command("configure").Flags(configurer).RunFunc(configurer.Run).
		Use("<container> <config-file> ...").
		Short("configure an existing container").
		Example("configure c1 demo.yaml")
	configOps := &ConfigOps{Client: client}
	cmd.Command("verify").Flags(configOps).RunFunc(configOps.Verify).
		Use("<config-file> ...").
		Short("verify config files").
		Example("verify *.yaml")
	cmd.Command("create-devices").Flags(configOps).RunFunc(configOps.CreateDevices).Use("{container-name} {configfile}...").Short("create devices")
	/* add devices:
	lxdops device add -p a.host -d /z/host/a -s 1 {configfile}...
	- create subdirectories
	- change ownership
	- add devices to profile, with optional suffix
	*/
	profile := cmd.Command("profile").Short("profile utilities")
	profileConfigurer := &ProfileConfigurer{Client: client}
	profile.Command("list").Flags(profileConfigurer).RunFunc(profileConfigurer.List).Short("list config profiles")
	profile.Command("diff").Flags(profileConfigurer).RunFunc(profileConfigurer.Diff).Short("compare container profiles with config")
	profile.Command("apply").Flags(profileConfigurer).RunFunc(profileConfigurer.Apply).Short("apply the config profiles to a container")
	profile.Command("reorder").Flags(profileConfigurer).RunFunc(profileConfigurer.Reorder).Short("reorder container profiles to match config order")

	lxdOps := &LxdOps{Client: client}
	profile.Command("exists").RunFunc(lxdOps.ProfileExists).Use("<profile>").Short("check if a profile exists")
	profile.Command("add-disk").RunFunc(lxdOps.AddDiskDevice).Use("<profile> <source> <path>").Short("add a disk device to a profile")
	cmd.Command("zfsroot").RunMethodE(lxdOps.ZFSRoot).Short("print zfs parent of lxd dataset")
	cmd.Command("pattern").RunFunc(lxdOps.Pattern).Short("run pattern substitution")

	parse := &ParseOp{}
	cmd.Command("parse").Flags(parse).RunFunc(parse.Run).
		Short("parse a config file").
		Use("<config-file>").
		Example("parse test.yaml")

	cmd.Command("description").RunFunc(configOps.Description).
		Short("print the description of a config file").
		Use("<config-file>").
		Example("test.yaml")

	networkOp := &NetworkOp{Client: client}
	cmd.Command("addresses").Flags(&networkOp).RunMethodE(networkOp.ExportAddresses).Short("export container addresses")

	containerOps := &ContainerOps{Client: client}
	containerCmd := cmd.Command("container")
	containerCmd.Command("profiles").RunFunc(containerOps.Profiles)
	containerCmd.Command("network").RunFunc(containerOps.Network)
	containerCmd.Command("wait").RunFunc(containerOps.Wait)
	containerCmd.Command("state").RunFunc(containerOps.State)

	projectOps := &ProjectOps{Client: client}
	projectCmd := cmd.Command("project").Short("project utilities")
	projectCmd.Command("create").Flags(projectOps).RunFunc(projectOps.Create)
	projectCmd.Command("copy-profiles").Flags(projectOps).RunFunc(projectOps.CopyProfiles)

	testCmd := cmd.Command("test")
	testCmd.Command("file").RunFunc(containerOps.File)
	testCmd.Command("push").RunFunc(containerOps.Push)
	testCmd.Command("project").RunFunc(projectOps.Use)

	usage.ApplyYaml(&cmd, usageData)

	return &cmd
}
