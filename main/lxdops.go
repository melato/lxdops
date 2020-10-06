package main

import (
	"melato.org/export/command"
	"melato.org/lxdops"
)

func main() {
	lxdops.OSTypes["alpine"] = &lxdops.OsTypeAlpine{}
	lxdops.OSTypes["debian"] = &lxdops.OsTypeDebian{}
	lxdops.OSTypes["ubuntu"] = &lxdops.OsTypeUbuntu{}
	cmd := lxdops.RootCommand()
	lxdops.AddShorewallCommands(cmd)
	command.Main(cmd)
}
