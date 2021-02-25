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
