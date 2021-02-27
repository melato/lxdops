package lxdops

import (
	"melato.org/script/v2"
)

type Snapshot struct {
	Client        *LxdClient `name:"-"`
	ConfigOptions ConfigOptions
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
}

func (t *Snapshot) Init() error {
	return t.ConfigOptions.Init()
}
func (t *Snapshot) Snapshot(configFile string, snapshot string) error {
	return t.ConfigOptions.Run([]string{configFile}, func(name string, config *Config) error {
		dev := NewDeviceConfigurer(t.Client, config)
		dev.DryRun = t.DryRun
		return dev.Snapshot(name, snapshot)
	})
}

func (t *DeviceConfigurer) Snapshot(name string, snapshot string) error {
	pattern := t.NewPattern(name)
	s := &script.Script{Trace: true, DryRun: t.DryRun}
	for _, fs := range t.Config.Filesystems {
		path, err := pattern.Substitute(fs.Pattern)
		if err != nil {
			return err
		}
		s.Run("sudo", "zfs", "snapshot", path+"@"+snapshot)
	}
	return s.Error()
}
