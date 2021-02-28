package main

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"melato.org/command"
	"melato.org/lxdops"
	"melato.org/lxdops/usage"
)

type Parse struct {
	File string `name:"i" usage:"input file"`
}

func (t *Parse) Init() error {
	t.File = "../commands.yaml"
	return nil
}

func Print(u *usage.Usage) error {
	data, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (t *Parse) Parse() error {
	if t.File == "" {
		return errors.New("missing input file")
	}
	var use usage.Usage
	data, err := os.ReadFile(t.File)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &use)
	if err != nil {
		return err
	}
	return Print(&use)
}

func Example() error {
	u1 := &usage.Usage{Usage: command.Usage{Short: "short1"}}
	u1.Commands = map[string]*usage.Usage{
		"b": &usage.Usage{Usage: command.Usage{Short: "b1"}},
		"c": &usage.Usage{Usage: command.Usage{Short: "b2"}},
	}
	return Print(u1)
}

func Extract() error {
	u := usage.Extract(lxdops.RootCommand())
	return Print(&u)
}

func main() {
	var cmd command.SimpleCommand
	parse := &Parse{}
	cmd.Command("parse").Flags(parse).RunFunc(parse.Parse)
	cmd.Command("example").RunFunc(Example)
	cmd.Command("extract").RunFunc(Extract)
	command.Main(&cmd)
}
