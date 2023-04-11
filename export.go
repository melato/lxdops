package lxdops

import (
	"path/filepath"

	"melato.org/script"
)

type ExportOps struct {
	ConfigOptions
	Dir    string `name:"d" usage:"import/export directory"`
	Image  bool   `name:"image" usage:"image too"`
	DryRun bool
}

func (t *ExportOps) Export(configFile string) error {
	instance, err := t.Instance(configFile)
	if err != nil {
		return err
	}
	dir := filepath.Join(t.Dir, instance.Name)
	filesystems, err := instance.Filesystems()
	if err != nil {
		return err
	}
	s := &script.Script{Trace: true, DryRun: t.DryRun}
	if t.Image {
		s.Cmd("lxc", "image", "export", instance.Name, filepath.Join(dir, instance.Name)).Run()
	}
	s.Cmd("sudo", "mkdir", "-p", dir).Run()

	for key, fs := range filesystems {
		tarFile := filepath.Join(dir, key+".tar.gz")
		s.Cmd("sudo", "tar", "cfz", tarFile, "-C", fs.Dir(), ".").Run()
	}
	return s.Error()
}

func (t *ExportOps) Import(configFile string) error {
	instance, err := t.Instance(configFile)
	if err != nil {
		return err
	}
	dir := filepath.Join(t.Dir, instance.Name)
	filesystems, err := instance.Filesystems()
	if err != nil {
		return err
	}
	s := &script.Script{Trace: true, DryRun: t.DryRun}
	if t.Image {
		s.Cmd("lxc", "image", "import", filepath.Join(dir, instance.Name+".tar.gz"), "--alias="+instance.Name).Run()
	}
	dev, err := NewDeviceConfigurer(instance)
	if err != nil {
		return err
	}
	dev.Trace = true
	dev.DryRun = t.DryRun
	err = dev.ConfigureDevices(instance)
	if err != nil {
		return err
	}

	for key, fs := range filesystems {
		tarFile := filepath.Join(dir, key+".tar.gz")
		s.Cmd("sudo", "tar", "xfz", tarFile, "-C", fs.Dir(), ".").Run()
	}
	return s.Error()
}
