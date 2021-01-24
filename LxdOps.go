package lxdops

import (
	"errors"
	"fmt"
	"os"

	"melato.org/lxdops/util"
	"melato.org/script"
)

type LxdOps struct {
	Trace bool `name:"trace,t" usage:"print exec arguments"`
}

func (t *LxdOps) ZFSRoot() error {
	path, err := ZFSRoot()
	if err == nil {
		fmt.Println(path)
	}
	return err
}

func (t *LxdOps) AddDiskDevice(args []string) error {
	if len(args) != 3 {
		return errors.New("Usage: <profile> <source> <path>")
	}
	profile := args[0]
	source := args[1]
	path := args[2]
	device := RandomDeviceName()
	script := &script.Script{Trace: t.Trace}
	script.Run("lxc", "profile", "device", "add", profile, device, "disk", "path="+path, "source="+source)
	return script.Error
}

func (t *LxdOps) ProfileExists(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: <profile>")
	}
	profile := args[0]
	script := &script.Script{Trace: t.Trace}
	cmd := script.Cmd("lxc", "profile", "get", profile, "x")
	cmd.Cmd.Stdout = &util.NullWriter{}
	cmd.MergeStderr()
	cmd.Run()
	if script.Error == nil {
		if t.Trace {
			fmt.Printf("profile %s exists\n", profile)
		}
		os.Exit(0)
	} else {
		if t.Trace {
			fmt.Printf("profile %s does not exist\n", profile)
		}
		os.Exit(1)
	}
	return script.Error
}

func (t *LxdOps) CurrentProject() error {
    project, err := CurrentProject()
    if err != nil {
        return err
    }
    fmt.Println(project)
    return nil
}
