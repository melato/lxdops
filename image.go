package lxdops

import (
	"path/filepath"

	"melato.org/script"
)

type ImageOps struct {
	ConfigOptions
	Dir    string
	FS     bool `name:"fs" usage:"import/export filesystems only"`
	DryRun bool
}

func (t *ImageOps) Export(configFile string) error {
	instance, err := t.Instance(configFile)
	if err != nil {
		return err
	}
	filesystems, err := instance.Filesystems()
	if err != nil {
		return err
	}
	s := &script.Script{Trace: true, DryRun: t.DryRun}
	if !t.FS {
		s.Cmd("lxc", "image", "export", instance.Name, filepath.Join(t.Dir, instance.Name)).Run()
	}
	for key, fs := range filesystems {
		tarFile := filepath.Join(t.Dir, key+".tar.gz")
		s.Cmd("sudo", "tar", "cfz", tarFile, "-C", fs.Dir(), ".").Run()
	}
	return s.Error()
}

func (t *ImageOps) Import(configFile string) error {
	instance, err := t.Instance(configFile)
	if err != nil {
		return err
	}
	filesystems, err := instance.Filesystems()
	if err != nil {
		return err
	}
	s := &script.Script{Trace: true, DryRun: t.DryRun}
	if !t.FS {
		s.Cmd("lxc", "image", "import", filepath.Join(t.Dir, instance.Name)+".tar.gz", "--alias="+instance.Name).Run()
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
		tarFile := filepath.Join(t.Dir, key+".tar.gz")
		s.Cmd("sudo", "tar", "xfz", tarFile, "-C", fs.Dir(), ".").Run()
	}
	return s.Error()
}
