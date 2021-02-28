// Package usage provides utilities for setting command usage from external or embedded files
package usage

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
	"melato.org/command"
)

// Usage provides a tree structure for command usage that parallels a command tree structure
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

// ApplyEnv looks for a file specified in an environment variable,
// reads this file, if it exists, reads this file and calls ApplyYaml with its content
// Returns true if it found usage data without errors.
// This way you can make changes to the usage data and see how it looks without recompiling.
// It prints any errors to stderr.
func ApplyEnv(cmd *command.SimpleCommand, envVar string) bool {
	file, env := os.LookupEnv(envVar)
	if env {
		if _, err := os.Stat(file); err == nil {
			fileContent, err := os.ReadFile(file)
			if err == nil {
				ApplyYaml(cmd, fileContent)
				return true
			}
			fmt.Fprintln(os.Stderr, err)
		}
	}
	return false
}

// ApplyYaml Extract usage from Yaml data and applies it to the command hierarchy.
// It prints any errors to stderr.
func ApplyYaml(cmd *command.SimpleCommand, yamlUsage []byte) bool {
	var use Usage
	err := yaml.Unmarshal(yamlUsage, &use)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	use.Apply(cmd)
	return true
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
