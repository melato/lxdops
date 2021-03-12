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
	cmd.Command("create-devices").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.CreateDevices, true))
	cmd.Command("create-profile").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.CreateProfile, true))

	snapshot := &Snapshot{}
	cmd.Command("snapshot").Flags(snapshot).RunFunc(snapshot.InstanceFunc(snapshot.Run, true))

	configurer := &Configurer{Client: client}
	cmd.Command("configure").Flags(configurer).RunFunc(configurer.InstanceFunc(configurer.ConfigureContainer, true))

	instanceOps := &InstanceOps{}
	instanceCmd := cmd.Command("instance").Flags(instanceOps)
	instanceCmd.Command("verify").RunFunc(instanceOps.InstanceFunc(instanceOps.Verify, true))
	instanceCmd.Command("description").RunFunc(instanceOps.InstanceFunc(instanceOps.Description, false))
	instanceCmd.Command("properties").RunFunc(instanceOps.InstanceFunc(instanceOps.Properties, false))
	instanceCmd.Command("filesystems").RunFunc(instanceOps.InstanceFunc(instanceOps.Filesystems, false))
	instanceCmd.Command("devices").RunFunc(instanceOps.InstanceFunc(instanceOps.Devices, false))
	instanceCmd.Command("project").RunFunc(instanceOps.InstanceFunc(instanceOps.Project, false))

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
	containerCmd.Command("config").RunFunc(containerOps.Config)
	containerCmd.Command("network").RunFunc(containerOps.Network)
	containerCmd.Command("wait").RunFunc(containerOps.Wait)
	containerCmd.Command("state").RunFunc(containerOps.State)
	containerDevices := &ContainerDevicesOp{ContainerOps: containerOps}
	containerCmd.Command("devices").Flags(containerDevices).RunFunc(containerDevices.Devices)

	projectCmd := cmd.Command("project")
	createProject := &ProjectCreate{Client: client}
	projectCmd.Command("create").Flags(createProject).RunFunc(createProject.Create)
	copyProfiles := &ProjectCopyProfiles{Client: client}
	projectCmd.Command("copy-profiles").Flags(copyProfiles).RunFunc(copyProfiles.CopyProfiles)

	usage.ApplyEnv(&cmd, "LXDOPS_USAGE", usageData)

	return &cmd
}
