package lxdops

import (
	"fmt"
	"path/filepath"
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

func (t *ParseOp) readIncludes(file string, included map[string]bool) error {
	config, err := ReadRawConfig(file)
	if err != nil {
		return err
	}
	dir := filepath.Dir(file)
	for _, include := range config.Include {
		path := include.Resolve(dir)
		if !included[string(path)] {
			fmt.Println(path)
			included[string(path)] = true
			t.readIncludes(string(path), included)
		}
	}
	return nil
}

func (t *ParseOp) Includes(file ...string) error {
	included := make(map[string]bool)
	for _, file := range file {
		if included[file] {
			continue
		}
		included[file] = true
		err := t.readIncludes(file, included)
		if err != nil {
			return err
		}
	}
	return nil
}
