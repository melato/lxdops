package main

import (
	"melato.org/export/command"
	"melato.org/lxdops"
	"melato.org/lxdops/os"
)

func main() {
	lxdops.OSTypes["alpine"] = &os.Alpine{}
	lxdops.OSTypes["debian"] = &os.Debian{}
	lxdops.OSTypes["ubuntu"] = &os.Ubuntu{}
	cmd := lxdops.RootCommand()
	lxdops.AddShorewallCommands(cmd)
	command.Main(cmd)
}
