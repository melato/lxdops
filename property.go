package lxdops

import (
	"fmt"
	"os"
	"path/filepath"

	"melato.org/lxdops/util"
)

type PropertyOptions struct {
	PropertiesFile   string            `name:"properties" usage:"a file containing global config properties.  Instance properties override global properties"`
	GlobalProperties map[string]string `name:"-"`
}

func (t *PropertyOptions) Init() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	t.PropertiesFile = filepath.Join(configDir, "lxdops", "properties.yaml")
	return nil
}

func (t *PropertyOptions) Configured() error {
	if t.PropertiesFile != "" {
		_, err := os.Stat(t.PropertiesFile)
		if err == nil {
			return util.ReadYaml(t.PropertiesFile, &t.GlobalProperties)
		}
	}
	return nil
}

func (t *PropertyOptions) List() {
	util.PrintMap(t.GlobalProperties)
}

func (t *PropertyOptions) File() {
	fmt.Println(t.PropertiesFile)
}

func (t *PropertyOptions) Set(key, value string) error {
	if t.GlobalProperties == nil {
		t.GlobalProperties = make(map[string]string)
	}
	t.GlobalProperties[key] = value
	if t.PropertiesFile != "" {
		dir := filepath.Dir(t.PropertiesFile)
		err := os.MkdirAll(dir, os.FileMode(0775))
		if err != nil {
			return err
		}
		return util.WriteYaml(t.PropertiesFile, t.GlobalProperties)
	}
	return nil
}
