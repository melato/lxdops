package lxdops

import (
	"errors"
	"fmt"
)

type ParseOp struct {
	Raw    bool   `usage:"do not process includes"`
	Script string `usage:"print the body of the script with the specified name"`
}

func (t *ParseOp) Run(args []string) error {
	var err error
	var config *Config
	if t.Raw {
		if len(args) != 1 {
			return errors.New("for raw config, please specify a single argument")
		}
		config, err = ReadRawConfig(args[0])
	} else {
		config, err = ReadConfigs(args...)
	}
	if err != nil {
		return err
	}
	if t.Script != "" {
		for _, script := range config.Scripts {
			fmt.Println("script", script.Name)
			if t.Script == script.Name {
				fmt.Println(script.Body)
			}
		}
	} else {
		config.Print()
	}
	return nil
}
