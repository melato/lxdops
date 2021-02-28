// Package usage provides utilities for setting command usage from external or embedded files
package usage

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
	"melato.org/command"
)

type Usage struct {
	command.Usage `yaml:",inline"`
	Commands      map[string]*Usage `yaml:"commands,omitempty"`
}

// Apply copies the usage to the command, recursively.
// Only non-empty fields are copied.
func (u *Usage) Apply(cmd *command.SimpleCommand) {
	if u.Short != "" {
		cmd.Short(u.Short)
	}
	if u.Use != "" {
		cmd.Use(u.Use)
	}
	if u.Long != "" {
		cmd.Long(u.Long)
	}
	if len(u.Examples) > 0 {
		cmd.Usage.Examples = u.Examples
	}
	commands := cmd.Commands()
	for name, c := range u.Commands {
		cmd, found := commands[name]
		if found {
			c.Apply(cmd)
		}
	}
}

// ApplyYaml Unmarshal usage from []byte and applies it to the command, recursively
func ApplyYaml(cmd *command.SimpleCommand, yamlUsage []byte) {
	var use Usage
	err := yaml.Unmarshal(yamlUsage, &use)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	use.Apply(cmd)
}

// Extract copies the usage fields from the command, recursively.
// It can be used to generate an external usage file from hardcoded usage strings
func Extract(cmd *command.SimpleCommand) Usage {
	var u Usage
	u.Usage = cmd.Usage
	for name, sub := range cmd.Commands() {
		if u.Commands == nil {
			u.Commands = make(map[string]*Usage)
		}
		su := Extract(sub)
		u.Commands[name] = &su
	}
	return u
}
