package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"melato.org/command"
	"melato.org/script"
)

type ApiBuild struct {
	TargetDir string
}

func (t *ApiBuild) Init() error {
	t.TargetDir = "../api/"
	return nil
}

func (t *ApiBuild) Run() error {
	gopath, found := os.LookupEnv("GOPATH")
	if !found {
		return errors.New("missing $GOPATH")
	}
	apiPath := "github.com/lxc/lxd/shared/api"
	srcDir := filepath.Join(gopath, "src", apiPath)
	script := script.Script{Trace: true}
	log, err := os.Create(filepath.Join(t.TargetDir, "log.txt"))
	if err != nil {
		return err
	}
	defer log.Close()
	for _, file := range []string{"project.go", "container.go", "container_state.go", "status_code.go", "container_snapshot.go", "container_backup.go"} {
		fmt.Fprintf(log, "%s/%s:\n", apiPath, file)
		script.Run("cp", filepath.Join(srcDir, file), t.TargetDir)
		script.Cmd("git", "log", "-1", "--date=iso", "--decorate=short", file).Dir(srcDir).
			ToWriter(log)
		fmt.Fprintln(log)
	}
	if script.Error != nil {
		return script.Error
	}
	err = log.Close()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var build ApiBuild
	var cmd command.SimpleCommand
	cmd.Flags(&build).RunMethodE(build.Run).Short("Copy dependent files from $GOPATH/src/github.com/lxc/lxd/shared/api")
	command.Main(&cmd)
}
