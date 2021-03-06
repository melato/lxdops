package lxdops

import (
	"path/filepath"
	"strings"

	"melato.org/lxdops/util"
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
	return InstanceFSList(list), nil
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
