package lxdops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"melato.org/table3"
)

type ParseOp struct {
	Raw     bool `usage:"do not process includes"`
	Verbose bool `name:"v" usage:"verbose"`
	//Script string `usage:"print the body of the script with the specified name"`
}

func (t *ParseOp) parseConfig(file string) (*Config, error) {
	if t.Raw {
		return ReadRawConfig(file)
	} else {
		r := &ConfigReader{Warn: true, Verbose: t.Verbose}
		return r.Read(file)
	}
}

func (t *ParseOp) Parse(file ...string) error {
	for _, file := range file {
		_, err := t.parseConfig(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *ParseOp) Print(file string) error {
	config, err := t.parseConfig(file)
	if err != nil {
		return err
	}
	config.Print()
	return nil
}

type ConfigOps struct {
}

func (t *ConfigOps) printScript(scripts []*Script, script string) {
	for _, s := range scripts {
		if script == s.Name {
			fmt.Println(s.Body)
		}
	}
}

func (t *ConfigOps) PrintProperties(file string) error {
	config, err := ReadConfig(file)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(config.Properties))
	for key, _ := range config.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	w := &table.FixedWriter{Writer: os.Stdout, NoHeaders: true}
	var key, value string
	w.Columns(
		table.NewColumn("property", func() interface{} { return key }),
		table.NewColumn("value", func() interface{} { return value }),
	)
	for _, key = range keys {
		value = config.Properties[key]
		w.WriteRow()
	}
	w.End()
	return nil
}

func (t *ConfigOps) Script(file string, script string) error {
	config, err := ReadConfig(file)
	if err != nil {
		return err
	}
	t.printScript(config.PreScripts, script)
	t.printScript(config.Scripts, script)
	return nil
}

func (t *ConfigOps) readIncludes(file string, included map[string]bool) error {
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

func (t *ConfigOps) Includes(file ...string) error {
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
