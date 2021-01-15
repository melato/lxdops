package lxdops

type DeviceCmd struct {
	Ops           *Ops
	DryRun        bool `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	ConfigOptions ConfigOptions
}

func (t *DeviceCmd) Init() error {
	t.Ops = &Ops{}
	return nil
}

func (t *DeviceCmd) Configured() error {
	if t.DryRun {
		t.Ops.Trace = true
	}
	return nil
}

func (t *DeviceCmd) Configure(name string, config *Config) error {
	dev := NewDeviceConfigurer(t.Ops)
	dev.SetDryRun(t.DryRun)
	return dev.ConfigureDevices(config, name)
}

func (t *DeviceCmd) Run(args []string) error {
	return t.ConfigOptions.Run(args, t.Configure)
}
