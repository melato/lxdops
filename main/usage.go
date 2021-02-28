package main

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"melato.org/command"
	"melato.org/lxdops/usage"
)

type Parse struct {
	File string `name:"i" usage:"input file"`
}

func (t *Parse) Init() error {
	t.File = "../usage.yaml"
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
	data, err = yaml.Marshal(&use)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func Example() error {
	u1 := &usage.Usage{Short: "short1"}
	u1.Commands = map[string]*usage.Usage{"b": &usage.Usage{Short: "b2"}}
	data, err := yaml.Marshal(u1)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func main() {
	var cmd command.SimpleCommand
	parse := &Parse{}
	cmd.Command("parse").Flags(parse).RunFunc(parse.Parse)
	cmd.Command("example").RunFunc(Example)
	command.Main(&cmd)
}
