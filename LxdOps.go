package lxdops

import (
	"errors"
	"fmt"
	"os"

	"melato.org/script"
)

type LxdOps struct {
	Ops *Ops
}

func (t *LxdOps) Init() error {
	t.Ops = &Ops{}
	return t.Ops.Init()
}

func (t *LxdOps) Configured() error {
	return t.Ops.Configured()
}

func (t *LxdOps) Verify(args []string) error {
	for _, configFile := range args {
		var err error
		var config *Config
		config, err = ReadConfig(configFile)
		isValid := false
		if err != nil {
			fmt.Println(err)
		}
		if err == nil {
			isValid = config.Verify()
		}
		fmt.Println(configFile, isValid)
	}
	return nil
}

func (t *LxdOps) ZFSRoot(args []string) error {
	path, err := t.Ops.ZFSRoot()
	if err == nil {
		fmt.Println(path)
	}
	return err
}

/** Print the description of a config file. */
func (t *LxdOps) Description(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: <config.yaml>")
	}
	var err error
	var config *Config
	config, err = ReadConfig(args[0])
	if err != nil {
		return err
	}
	fmt.Println(config.Description)
	return nil
}

func (t *LxdOps) AddDiskDevice(args []string) error {
	if len(args) != 3 {
		return errors.New("Usage: <profile> <source> <path>")
	}
	profile := args[0]
	source := args[1]
	path := args[2]
	device := RandomDeviceName()
	script := &script.Script{Trace: t.Ops.Trace}
	script.Run("lxc", "profile", "device", "add", profile, device, "disk", "path="+path, "source="+source)
	return script.Error
}

func (t *LxdOps) ProfileExists(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: <profile>")
	}
	profile := args[0]
	script := &script.Script{Trace: t.Ops.Trace}
	cmd := script.Cmd("lxc", "profile", "get", profile, "x")
	cmd.Cmd.Stdout = &NullWriter{}
	cmd.MergeStderr()
	cmd.Run()
	if script.Error == nil {
		if t.Ops.Trace {
			fmt.Printf("profile %s exists\n", profile)
		}
		os.Exit(0)
	} else {
		if t.Ops.Trace {
			fmt.Printf("profile %s does not exist\n", profile)
		}
		os.Exit(1)
	}
	return script.Error
}
