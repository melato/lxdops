package lxdops

import (
	"os"
	"path/filepath"

	"melato.org/export/program"
)

type HostConfigurer struct {
	Ops    *Ops `name:""`
	DryRun bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	prog   program.Params
}

func (t *HostConfigurer) Configured() error {
	t.prog.DryRun = t.DryRun
	t.prog.Trace = t.Ops.Trace
	return nil
}

/** copy certain host files to a shared directory where the guest can see them:
  authorized_keys
  hostname
*/
func (t *HostConfigurer) copyHostInfo() error {
	var err error
	path, err := t.Ops.GetPath("opt")
	if err != nil {
		return err
	}
	dir := filepath.Join(path, "host")
	sudo := false
	err = program.NewProgram("mkdir").Sudo(sudo).Run("-p", dir)
	if err != nil {
		return err
	}
	err = program.NewProgram("cp").Sudo(sudo).Run(os.ExpandEnv("${HOME}/.ssh/authorized_keys"), dir)
	if err != nil {
		return err
	}
	err = program.NewProgram("cp").Sudo(sudo).Run("/etc/hostname", dir+"/name")
	if err != nil {
		return err
	}
	return nil
}

func (t *HostConfigurer) RunE() error {
	err := t.copyHostInfo()
	if err != nil {
		return err
	}
	opt, err := t.Ops.GetPath("opt")
	if err != nil {
		return err
	}
	t.prog.NewProgram("mkdir").Sudo(true).Run("-p", filepath.Join(opt, "ubuntu"))
	lsb, err := ReadProperties("/etc/lsb-release")
	if err != nil {
		return err
	}
	release := lsb["DISTRIB_RELEASE"]
	if release != "" {
		file := filepath.Join(opt, "ubuntu", "ubuntu-"+release+".list")
		err = t.prog.NewProgram("cp").Sudo(true).Run("/etc/apt/sources.list", file)
	}
	return err
}
