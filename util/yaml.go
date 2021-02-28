package util

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func PrintYaml(v interface{}) {
	data, err := yaml.Marshal(v)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Stdout.Write(data)
	fmt.Println()
}

func ReadYaml(file string, v interface{}) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, v)
}
