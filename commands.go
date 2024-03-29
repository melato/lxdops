package lxdops

import (
	_ "embed"

	"melato.org/command"
	"melato.org/command/usage"
	"melato.org/lxdops/lxdutil"
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
	client := &lxdutil.LxdClient{}
	var cmd command.SimpleCommand
	cmd.Flags(client)
	launcher := &Launcher{Client: client}
	cmd.Command("launch").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.LaunchContainer, true))
	cmd.Command("delete").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.DeleteContainer, false))
	cmd.Command("destroy").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.DestroyContainer, false))
	cmd.Command("rebuild").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.Rebuild, true))
	cmd.Command("rename").Flags(launcher).RunFunc(launcher.Rename)
	cmd.Command("create-devices").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.CreateDevices, true))
	cmd.Command("create-profile").Flags(launcher).RunFunc(launcher.InstanceFunc(launcher.CreateProfile, false))

	snapshot := &Snapshot{}
	cmd.Command("snapshot").Flags(snapshot).RunFunc(snapshot.InstanceFunc(snapshot.Run, false))

	rollback := &Rollback{}
	cmd.Command("rollback").Flags(rollback).RunFunc(rollback.InstanceFunc(rollback.Run, false))

	configurer := &Configurer{Client: client}
	cmd.Command("configure").Flags(configurer).RunFunc(configurer.InstanceFunc(configurer.ConfigureContainer, false))

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
	profile.Command("list").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.List, false))
	profile.Command("diff").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.Diff, false))
	profile.Command("apply").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.Apply, false))
	profile.Command("reorder").Flags(profileConfigurer).RunFunc(profileConfigurer.InstanceFunc(profileConfigurer.Reorder, false))
	profileOps := &lxdutil.ProfileOps{Client: client}
	profile.Command("export").Flags(profileOps).RunFunc(profileOps.Export)
	profile.Command("import").Flags(profileOps).RunFunc(profileOps.Import)

	lxdOps := &LxdOps{Client: client}
	profile.Command("exists").RunFunc(lxdOps.ProfileExists)
	addDisk := &AddDisk{Client: client}
	profile.Command("add-disk").Flags(addDisk).RunFunc(addDisk.Add)

	propertyOps := &PropertyOptions{}
	propertyCmd := cmd.Command("property").Flags(propertyOps)
	propertyCmd.Command("list").RunFunc(propertyOps.List)
	propertyCmd.Command("set").RunFunc(propertyOps.Set)
	propertyCmd.Command("get").RunFunc(propertyOps.Get)
	propertyCmd.Command("file").RunFunc(propertyOps.File)

	configCmd := cmd.Command("config")
	parse := &ParseOp{}
	configCmd.Command("parse").Flags(parse).RunFunc(parse.Parse)
	configCmd.Command("print").Flags(parse).RunFunc(parse.Print)
	configOps := &ConfigOps{}
	configCmd.Command("properties").RunFunc(configOps.PrintProperties)
	configCmd.Command("includes").RunFunc(configOps.Includes)
	configCmd.Command("script").RunFunc(configOps.Script)

	containerOps := &lxdutil.InstanceOps{Client: client}
	containerCmd := cmd.Command("container")
	containerCmd.Flags(containerOps)
	containerCmd.Command("profiles").RunFunc(containerOps.Profiles)
	containerCmd.Command("info").RunFunc(containerOps.Info)
	containerCmd.Command("config").RunFunc(containerOps.Config)
	containerCmd.Command("network").RunFunc(containerOps.Network)
	containerCmd.Command("wait").RunFunc(containerOps.Wait)
	containerCmd.Command("state").RunFunc(containerOps.State)
	containerDevices := &lxdutil.InstanceDevicesOp{InstanceOps: containerOps}
	containerCmd.Command("devices").Flags(containerDevices).RunFunc(containerDevices.Devices)
	containerCmd.Command("statistics").RunFunc(containerOps.Statistics)
	containerCmd.Command("hwaddr").RunFunc(containerOps.ListHwaddr)
	containerCmd.Command("images").RunFunc(containerOps.ListImages)

	templateOps := &lxdutil.TemplateOps{Client: client}
	cmd.Command("template").Flags(templateOps).RunFunc(templateOps.Apply)

	networkOp := &lxdutil.NetworkOp{Client: client}
	containerCmd.Command("addresses").Flags(networkOp).RunFunc(networkOp.ExportAddresses)

	numberOp := &lxdutil.AssignNumbers{Client: client}
	containerCmd.Command("number").Flags(numberOp).RunFunc(numberOp.Run)

	projectCmd := cmd.Command("project")
	createProject := &lxdutil.ProjectCreate{Client: client}
	projectCmd.Command("create").Flags(createProject).RunFunc(createProject.Create)
	copyProfiles := &lxdutil.ProjectCopyProfiles{Client: client}
	projectCmd.Command("copy-profiles").Flags(copyProfiles).RunFunc(copyProfiles.CopyProfiles)

	exportOps := &ExportOps{}
	cmd.Command("export").Flags(exportOps).RunFunc(exportOps.Export)
	cmd.Command("import").Flags(exportOps).RunFunc(exportOps.Import)

	var migrate Migrate
	cmd.Command("copy-filesystems").Flags(&migrate).RunFunc(migrate.CopyFilesystems)

	imageOps := &lxdutil.ImageOps{Client: client}
	imageCmd := cmd.Command("image")
	imageCmd.Flags(imageOps)
	imageCmd.Command("list").RunFunc(imageOps.List)
	imageCmd.Command("fingerprints").RunFunc(imageOps.ListFingerprints)
	imageCmd.Command("containers").RunFunc(imageOps.Containers)

	usage.Apply(&cmd, usageData)

	return &cmd
}
