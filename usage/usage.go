package usage

import (
	"melato.org/command"
)

type Usage struct {
	Short    string            `yaml:"short"`
	Long     string            `yaml:"long"`
	Use      string            `yaml:"use"`
	Example  []string          `yaml:"example"`
	Commands map[string]*Usage `yaml:"commands"`
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
	for _, example := range u.Example {
		cmd.Example(example)
	}
}
