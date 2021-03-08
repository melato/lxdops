package lxdops

import (
	"path/filepath"
	"strings"

	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type Instance struct {
	Config     *Config
	Name       string
	Properties *util.PatternProperties
	fspaths    map[string]InstanceFS
}

func (config *Config) NewInstance(name string) *Instance {
	instance := &Instance{Config: config, Name: name}
	instance.Properties = config.NewProperties(name)
	return instance
}

func (t *Instance) ProfileName() (string, error) {
	if t.Config.Profile != "" {
		return t.Config.Profile.Substitute(t.Properties)
	}
	return t.Name + "." + DefaultProfileSuffix, nil
}

func (t *Instance) Container() string {
	return t.Name
}

func (t *Instance) Filesystems() (map[string]InstanceFS, error) {
	if t.fspaths == nil {
		fspaths := make(map[string]InstanceFS)
		for id, fs := range t.Config.Filesystems {
			path, err := fs.Pattern.Substitute(t.Properties)
			if err != nil {
				return nil, err
			}
			fspaths[id] = InstanceFS{Id: id, Path: path, Filesystem: fs}
		}
		t.fspaths = fspaths
	}
	return t.fspaths, nil
}

func (t *Instance) FilesystemList() ([]InstanceFS, error) {
	paths, err := t.Filesystems()
	if err != nil {
		return nil, err
	}
	var list []InstanceFS
	for _, path := range paths {
		list = append(list, path)
	}
	InstanceFSList(list).Sort()
	return list, nil
}

func (t *Instance) DeviceList() ([]InstanceDevice, error) {
	var devices []InstanceDevice
	for name, device := range t.Config.Devices {
		d := InstanceDevice{Name: name, Device: device}
		dir, err := t.DeviceDir(name, device)
		if err != nil {
			return nil, err
		}
		d.Source = dir
		devices = append(devices, d)
	}

	InstanceDeviceList(devices).Sort()
	return devices, nil
}

func (t *Instance) DeviceDir(deviceId string, device *Device) (string, error) {
	dir, err := device.Dir.Substitute(t.Properties)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(dir, "/") {
		return dir, nil
	}
	if dir == "" {
		dir = deviceId
	} else if device.Dir == "." {
		dir = ""
	}

	fspaths, err := t.Filesystems()
	if err != nil {
		return "", err
	}
	fsPath, exists := fspaths[device.Filesystem]
	if !exists {
		return "", nil
	}

	if dir != "" {
		return filepath.Join(fsPath.Dir(), dir), nil
	} else {
		return fsPath.Dir(), nil
	}
}

// SourceName returns the instance name of the source config, if any.
func (t *Instance) SourceName() string {
	config := t.Config
	if config.SourceConfig == "" {
		return ""
	}
	return BaseName(string(config.SourceConfig))
}

// SourceContainer returns the name of the container of the source config, if any.
func (t *Instance) SourceContainer() (string, error) {
	config := t.Config
	if config.SourceConfig == "" {
		return "", nil
	}
	sourceConfig, err := config.GetSourceConfig()
	if err != nil {
		return "", err
	}
	sourceInstance := sourceConfig.NewInstance(BaseName(string(config.SourceConfig)))
	return sourceInstance.Container(), nil
}

// Snapshot creates a snapshot of all ZFS filesystems of the instance
func (instance *Instance) Snapshot(t SnapshotParams) error {
	filesystems, err := instance.FilesystemList()
	if err != nil {
		return err
	}
	fslist := InstanceFSList(filesystems)
	fslist.Sort()
	s := &script.Script{Trace: true}
	if t.Destroy {
		if t.Recursive {
			roots := fslist.Roots()
			for _, fs := range roots {
				s.Run("sudo", "zfs", "destroy", "-R", fs.Path+"@"+t.Snapshot)
			}
		} else {
			for _, fs := range fslist {
				s.Run("sudo", "zfs", "destroy", fs.Path+"@"+t.Snapshot)
			}
		}
	} else {
		for _, fs := range fslist {
			s.Run("sudo", "zfs", "snapshot", fs.Path+"@"+t.Snapshot)
		}
	}
	return s.Error()
}
