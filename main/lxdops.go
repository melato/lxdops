package main

import (
	"melato.org/export/command"
	"melato.org/lxdops"
)

func main() {
	command.Main(&lxdops.RootCommand{})
}
