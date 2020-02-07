package lxdops

import (
	"melato.org/export/command"
)

type ConfigureOp struct {
	command.Base
	Configurer *Configurer
}

func (t *ConfigureOp) Configured() error {
	return t.Configurer.Configured()
}

func (t *ConfigureOp) Run(args []string) error {
	return t.Configurer.Run(args)
}

func (t *ConfigureOp) Usage() *command.Usage {
	return &command.Usage{
		Use:     "<container> <config-file> ...",
		Short:   "configure an existing container",
		Example: "ops configure c1 demo.yaml",
	}
}
