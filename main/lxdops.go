package main

import (
	"melato.org/export/command"
	"melato.org/lxdops"
)

func main() {
	cmd := lxdops.RootCommand()
	lxdops.AddShorewallCommands(cmd)
	command.Main(cmd)
}
