package lxdops

import (
	"melato.org/export/command"
)

type LaunchOp struct {
	command.Base
	Launcher *Launcher
}

func (t *LaunchOp) Init() error {
	return t.Launcher.Init()
}

func (t *LaunchOp) Configured() error {
	return t.Launcher.Configured()
}

func (t *LaunchOp) Run(args []string) error {
	return t.Launcher.Run(args)
}

func (op *LaunchOp) Usage() *command.Usage {
	return &command.Usage{
		Use:     "<container> <config-file> ...",
		Short:   "launch a container",
		Example: "ops launch php php.yaml",
	}
}
