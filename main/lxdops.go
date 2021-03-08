package main

import (
	_ "embed"
	"fmt"

	"melato.org/command"
	"melato.org/lxdops"
	"melato.org/lxdops/os"
)

//go:embed version
var version string

func main() {
	lxdops.OSTypes["alpine"] = &os.Alpine{}
	lxdops.OSTypes["debian"] = &os.Debian{}
	lxdops.OSTypes["ubuntu"] = &os.Ubuntu{}
	cmd := lxdops.RootCommand()
	cmd.Command("version").NoConfig().RunMethod(func() { fmt.Println(version) }).Short("print program version")
	command.Main(cmd)
}
