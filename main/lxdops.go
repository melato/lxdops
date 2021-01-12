package main

import (
	"fmt"

	"melato.org/command"
	"melato.org/lxdops"
	"melato.org/lxdops/os"
)

func main() {
	lxdops.OSTypes["alpine"] = &os.Alpine{}
	lxdops.OSTypes["debian"] = &os.Debian{}
	lxdops.OSTypes["ubuntu"] = &os.Ubuntu{}
	cmd := lxdops.RootCommand()
	cmd.Command("version").RunMethod(func() { fmt.Println(Version) }).Short("print program version")
	command.Main(cmd)
}
