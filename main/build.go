package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"melato.org/command"
	"melato.org/script"
)

type Build struct {
	Api bool `usage:"copy api files from $GOPATH/src/github.com/lxc/lxd/shared/api/`
}

func (t *Build) GenerateVersion() error {
	script := script.Script{Trace: true}
	script.Run("mkversion", "-t", "version.tpl", "version.go")
	if script.Error != nil {
		fmt.Println(script.Error)
		script.Error = nil
		fmt.Println("using unknown version")
		script.Run("cp", "unknown_version.go", "version.go")
	}
	return script.Error
}

func (t *Build) Compile() error {
	err := t.GenerateVersion()
	if err != nil {
		return err
	}
	script := script.Script{Trace: true}
	cmd := script.Cmd("go", "install",
		"-ldflags", `-extldflags "-static"`,
		"lxdops.go", "version.go")
	cmd.Cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Run()
	return script.Error
}

func (t *Build) CopyAPI() error {
	gopath, found := os.LookupEnv("GOPATH")
	if !found {
		return errors.New("missing $GOPATH")
	}
	srcDir := filepath.Join(gopath, "src/github.com/lxc/lxd/shared/api")
	dstDir := "../lxd/"
	script := script.Script{Trace: true}
	for _, file := range []string{"project.go", "container.go", "container_state.go"} {
		script.Run("cp", filepath.Join(srcDir, file), dstDir)
		script.Cmd("git", "log", "-1", "--date=iso", "--decorate=short", file).Dir(srcDir).
			ToFile(filepath.Join(dstDir, "commit-"+file+".txt"))
	}
	return script.Error
}

func (t *Build) Run() error {
	if t.Api {
		return t.CopyAPI()
	}
	return t.Compile()
}

func main() {
	var build Build
	var cmd command.SimpleCommand
	cmd.Flags(&build).RunMethodE(build.Run)
	command.Main(&cmd)
}
