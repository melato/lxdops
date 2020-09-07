package lxdops

import (
	"fmt"

	"melato.org/export/command"
)

type VerifyOp struct {
}

func (t *VerifyOp) Run(args []string) error {
	for _, configFile := range args {
		var err error
		var config *Config
		config, err = ReadConfig(configFile)
		isValid := false
		if err != nil {
			fmt.Println(err)
		}
		if err == nil {
			isValid = config.Verify()
		}
		fmt.Println(configFile, isValid)
	}
	return nil
}

func (op *VerifyOp) Usage() *command.Usage {
	return &command.Usage{
		Short:   "verify config files",
		Use:     "<config-file>...",
		Example: "ops verify *.xml",
	}
}
