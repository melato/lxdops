package lxdops

import (
	"fmt"
)

type ParseOp struct {
	Raw    bool   `usage:"do not process includes"`
	Script string `usage:"print the body of the script with the specified name"`
}

func (t *ParseOp) Run(file string) error {
	var err error
	var config *Config
	if t.Raw {
		config, err = ReadRawConfig(file)
	} else {
		config, err = ReadConfig(file)
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
