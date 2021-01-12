package main

import (
	"fmt"
	"os"

	"melato.org/command"
	"melato.org/script"
)

func Build() error {
	script := script.Script{Trace: true}
	script.Run("mkversion", "-t", "version.tpl", "version.go")
	if script.Error != nil {
		fmt.Println(script.Error)
		script.Error = nil
		fmt.Println("using unknown version")
		script.Run("cp", "unknown_version.go", "version.go")
	}
	cmd := script.Cmd("go", "install",
		"-ldflags", `-extldflags "-static"`,
		"lxdops.go", "version.go")
	cmd.Cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Run()
	return script.Error
}

func main() {
	cmd := &command.SimpleCommand{}
	cmd.RunMethodE(Build)
	command.Main(cmd)
}
