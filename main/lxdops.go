package main

import (
	_ "embed"
	"fmt"

	"melato.org/cloudconfig/ostype"
	"melato.org/command"
	"melato.org/lxdops"
)

//go:embed version
var version string

func main() {
	lxdops.OSTypes["alpine"] = &ostype.Alpine{}
	lxdops.OSTypes["debian"] = &ostype.Debian{}
	lxdops.OSTypes["ubuntu"] = &ostype.Debian{}
	cmd := lxdops.RootCommand()
	cmd.Command("version").NoConfig().RunMethod(func() { fmt.Println(version) }).Short("print program version")
	command.Main(cmd)
}
