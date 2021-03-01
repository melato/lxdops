package lxdops

import (
	_ "embed"

	"melato.org/command"
	"melato.org/command/usage"
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
	cmd.Command("launch").Flags(launcher).RunFunc(launcher.Launch)
	cmd.Command("delete").Flags(launcher).RunFunc(launcher.Delete)
	cmd.Command("rebuild").Flags(launcher).RunFunc(launcher.Rebuild)
	cmd.Command("rename").Flags(launcher).RunFunc(launcher.Rename)
	cmd.Command("filesystems").Flags(launcher).RunFunc(launcher.PrintFilesystems)
	cmd.Command("devices").Flags(launcher).RunFunc(launcher.PrintDevices)

	snapshot := &Snapshot{Client: client}
	cmd.Command("snapshot").Flags(snapshot).RunFunc(snapshot.Snapshot)

	configurer := &Configurer{Client: client}
	cmd.Command("configure").Flags(configurer).RunFunc(configurer.Run)
	configOps := &ConfigOps{Client: client}
	cmd.Command("verify").Flags(configOps).RunFunc(configOps.Verify)
	cmd.Command("create-devices").Flags(configOps).RunFunc(configOps.CreateDevices)
	/* add devices:
	lxdops device add -p a.host -d /z/host/a -s 1 {configfile}...
	- create subdirectories
	- change ownership
	- add devices to profile, with optional suffix
	*/
	profile := cmd.Command("profile")
	profileConfigurer := &ProfileConfigurer{Client: client}
	profile.Command("list").Flags(profileConfigurer).RunFunc(profileConfigurer.List)
	profile.Command("diff").Flags(profileConfigurer).RunFunc(profileConfigurer.Diff)
	profile.Command("apply").Flags(profileConfigurer).RunFunc(profileConfigurer.Apply)
	profile.Command("reorder").Flags(profileConfigurer).RunFunc(profileConfigurer.Reorder)

	lxdOps := &LxdOps{Client: client}
	profile.Command("exists").RunFunc(lxdOps.ProfileExists)
	profile.Command("add-disk").RunFunc(lxdOps.AddDiskDevice)
	cmd.Command("zfsroot").RunMethodE(lxdOps.ZFSRoot)
	cmd.Command("pattern").RunFunc(lxdOps.Pattern)

	parse := &ParseOp{}
	cmd.Command("parse").Flags(parse).RunFunc(parse.Run)
	cmd.Command("description").RunFunc(configOps.Description)

	networkOp := &NetworkOp{Client: client}
	cmd.Command("addresses").Flags(networkOp).RunMethodE(networkOp.ExportAddresses)

	containerOps := &ContainerOps{Client: client}
	containerCmd := cmd.Command("container")
	containerCmd.Command("profiles").RunFunc(containerOps.Profiles)
	containerCmd.Command("network").RunFunc(containerOps.Network)
	containerCmd.Command("wait").RunFunc(containerOps.Wait)
	containerCmd.Command("state").RunFunc(containerOps.State)

	projectOps := &ProjectOps{Client: client}
	projectCmd := cmd.Command("project")
	projectCmd.Command("create").Flags(projectOps).RunFunc(projectOps.Create)
	projectCmd.Command("copy-profiles").Flags(projectOps).RunFunc(projectOps.CopyProfiles)

	testCmd := cmd.Command("test")
	testCmd.Command("file").RunFunc(containerOps.File)
	testCmd.Command("push").RunFunc(containerOps.Push)
	testCmd.Command("project").RunFunc(projectOps.Use)

	_ = usage.ApplyEnv(&cmd, "USAGE_FILE") || usage.ApplyYaml(&cmd, usageData)

	return &cmd
}
