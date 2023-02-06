package lxdops

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

/** Read raw config from yaml */
func ReadConfigYaml(file string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var x Config
	err = yaml.Unmarshal(data, &x)
	if err != nil {
		return nil, err
	}
	return &x, err
}

func PrintConfigYaml(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
