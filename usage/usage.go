// Package usage provides utilities for setting command usage from external or embedded files
package usage

import (
	"melato.org/command"
)

type Usage struct {
	command.Usage `yaml:",inline"`
	Commands      map[string]*Usage `yaml:"commands,omitempty"`
}

func (t *Usage) Get(command ...string) *Usage {
	u := t
	for _, c := range command {
		var found bool
		u, found = u.Commands[c]
		if !found {
			return &Usage{}
		}
	}
	return u
}

// Apply copies the usage fields to the command, if they are not empty.
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
