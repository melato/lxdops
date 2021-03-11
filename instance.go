package lxdops

import (
	"errors"
	"path/filepath"
	"strings"

	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type Instance struct {
	Config       *Config
	Name         string
	container    string
	profile      string
	origin       *Origin
	deviceSource *DeviceSource
	Project      string
	Properties   *util.PatternProperties
	fspaths      map[string]InstanceFS
	sourceConfig *Config
}

type Origin struct {
	Project   string
	Container string
	Snapshot  string
}

type DeviceSource struct {
	Instance *Instance
	Snapshot string
	Clone    bool
}

func (t *DeviceSource) IsDefined() bool {
	return t.Instance != nil
}

func (t *Origin) parse(s string) {
	i := strings.Index(s, "/")
	pc := s
	if i >= 0 {
		t.Snapshot = s[i+1:]
		pc = s[0:i]
	}
	i = strings.Index(pc, "_")
	if i >= 0 {
		t.Project = s[0:i]
		t.Container = s[i+1:]
	} else {
		t.Container = pc
	}
}

func (t *Origin) IsDefined() bool {
	return t.Container != ""
}

func (config *Config) NewInstance(name string) (*Instance, error) {
	t := &Instance{Config: config, Name: name}
	t.Properties = config.NewProperties(name)
	t.container = name
	var err error
	if config.Profile != "" {
		t.profile, err = config.Profile.Substitute(t.Properties)
		if err != nil {
			return nil, err
		}
	} else {
		t.profile = t.Name + "." + DefaultProfileSuffix
	}
	return t, nil
}

func (t *Instance) GetOrigin() (*Origin, error) {
	if t.origin == nil {
		s, err := t.Config.Origin.Substitute(t.Properties)
		if err != nil {
			return nil, err
		}
		origin := &Origin{}
		if s != "" {
			origin.parse(s)
			if !origin.IsDefined() {
				origin, err = t.sourceOrigin()
				if err != nil {
					return nil, err
				}
				if origin == nil {
					return nil, errors.New("missing container origin")
				}
			}
			if origin.Project == "" {
				origin.Project = t.Project
			}
		}
		t.origin = origin
	}
	return t.origin, nil
}

func (t *Instance) newDeviceSource() (*DeviceSource, error) {
	config := t.Config
	if config.DeviceTemplate != "" && config.DeviceOrigin != "" {
		return nil, errors.New("using both device-template and device-origin is not allowed")
	}
	source := &DeviceSource{}
	var name string
	if config.DeviceTemplate != "" {
		var err error
		name, err = config.DeviceTemplate.Substitute(t.Properties)
		if err != nil {
			return nil, err
		}
	} else if config.DeviceOrigin != "" {
		s, err := config.DeviceTemplate.Substitute(t.Properties)
		if err != nil {
			return nil, err
		}
		parts := strings.Split(s, "@")
		if len(parts) != 2 || len(parts[1]) == 0 {
			return nil, errors.New("missing device origin snapshot: " + s)
		}
		name = parts[0]
		source.Snapshot = parts[1]
	} else {
		return source, nil
	}
	if name == "" && config.SourceConfig == "" {
		return nil, errors.New("missing devices source name")
	}
	sourceConfig, err := t.GetSourceConfig()
	if err != nil {
		return nil, err
	}
	source.Instance, err = sourceConfig.NewInstance(name)
	if err != nil {
		return nil, err
	}
	return source, nil
}

func (t *Instance) GetDeviceSource() (*DeviceSource, error) {
	if t.deviceSource == nil {
		var err error
		t.deviceSource, err = t.newDeviceSource()
		if err != nil {
			return nil, err
		}
	}
	return t.deviceSource, nil
}

func (t *Instance) ProfileName() string {
	return t.profile
}

func (t *Instance) Container() string {
	return t.container
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
func (t *Instance) sourceOrigin() (*Origin, error) {
	config := t.Config
	if config.SourceConfig == "" {
		return nil, nil
	}
	sourceConfig, err := t.GetSourceConfig()
	if err != nil {
		return nil, err
	}
	sourceInstance, err := sourceConfig.NewInstance(BaseName(string(config.SourceConfig)))
	if err != nil {
		return nil, err
	}
	origin := &Origin{Project: sourceInstance.Project, Container: sourceInstance.Container(), Snapshot: sourceConfig.Snapshot}
	return origin, nil
}

// Snapshot creates a snapshot of all ZFS filesystems of the instance
func (instance *Instance) Snapshot(name string) error {
	filesystems, err := instance.FilesystemList()
	if err != nil {
		return err
	}
	s := &script.Script{Trace: true}
	for _, fs := range filesystems {
		s.Run("sudo", "zfs", "snapshot", fs.Path+"@"+name)
	}
	return s.Error()
}

// GetSourceConfig returns the parsed configuration specified by Config.SourceConfig
// If there is no Config.SourceConfig, it returns this instance's config
// It returns a non nil *Config or an error.
func (t *Instance) GetSourceConfig() (*Config, error) {
	if t.Config.SourceConfig == "" {
		return t.Config, nil
	}
	if t.sourceConfig == nil {
		config, err := ReadConfig(string(t.Config.SourceConfig))
		if err != nil {
			return nil, err
		}
		t.sourceConfig = config
	}
	return t.sourceConfig, nil
}
