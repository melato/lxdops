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
	cmd.Command("launch").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.LaunchContainer, true))
	cmd.Command("delete").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.DeleteContainer, true))
	cmd.Command("rebuild").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.Rebuild, true))
	cmd.Command("rename").Flags(launcher).RunFunc(launcher.Rename)
	cmd.Command("create-devices").Flags(launcher).RunFunc(launcher.CreateDevices)

	snapshot := &Snapshot{Client: client}
	cmd.Command("snapshot").Flags(snapshot).RunFunc(snapshot.Snapshot)

	configurer := &Configurer{Client: client}
	cmd.Command("configure").Flags(configurer).RunFunc(configurer.InstanceFunc(configurer.ConfigureContainer, true))

	configOps := &ConfigOps{}
	cmd.Command("verify").Flags(configOps).RunFunc(configOps.InstanceFunc(configOps.Verify, true))
	cmd.Command("description").RunFunc(configOps.InstanceFunc(configOps.Description, false))
	cmd.Command("properties").Flags(configOps).RunFunc(configOps.InstanceFunc(configOps.Properties, false))
	cmd.Command("filesystems").Flags(configOps).RunFunc(configOps.InstanceFunc(configOps.PrintFilesystems, false))
	cmd.Command("devices").Flags(configOps).RunFunc(configOps.InstanceFunc(configOps.PrintDevices, false))

	profile := cmd.Command("profile")
	profileConfigurer := &ProfileConfigurer{Client: client}
	profile.Command("list").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.List, true))
	profile.Command("diff").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.Diff, true))
	profile.Command("apply").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.Apply, true))
	profile.Command("reorder").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.Reorder, true))

	lxdOps := &LxdOps{Client: client}
	profile.Command("exists").RunFunc(lxdOps.ProfileExists)
	profile.Command("add-disk").RunFunc(lxdOps.AddDiskDevice)
	cmd.Command("zfsroot").RunMethodE(lxdOps.ZFSRoot)

	parse := &ParseOp{}
	cmd.Command("parse").Flags(parse).RunFunc(parse.Run)

	networkOp := &NetworkOp{Client: client}
	cmd.Command("addresses").Flags(networkOp).RunMethodE(networkOp.ExportAddresses)

	containerOps := &ContainerOps{Client: client}
	containerCmd := cmd.Command("container")
	containerCmd.Flags(containerOps)
	containerCmd.Command("profiles").RunFunc(containerOps.Profiles)
	containerCmd.Command("network").RunFunc(containerOps.Network)
	containerCmd.Command("wait").RunFunc(containerOps.Wait)
	containerCmd.Command("state").RunFunc(containerOps.State)

	projectCmd := cmd.Command("project")
	createProject := &ProjectCreate{Client: client}
	projectCmd.Command("create").Flags(createProject).RunFunc(createProject.Create)
	copyProfiles := &ProjectCopyProfiles{Client: client}
	projectCmd.Command("copy-profiles").Flags(copyProfiles).RunFunc(copyProfiles.CopyProfiles)

	usage.ApplyEnv(&cmd, "LXDOPS_USAGE", usageData)

	return &cmd
}
