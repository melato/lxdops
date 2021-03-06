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
	fspaths    map[string]FSPath
}

func (config *Config) NewInstance(name string) (*Instance, error) {
	instance := &Instance{Config: config, Name: name}
	instance.Properties = config.NewProperties(name)
	return instance, nil
}

func (t *Instance) FilesystemPaths() (map[string]FSPath, error) {
	if t.fspaths == nil {
		fspaths := make(map[string]FSPath)
		for _, fs := range t.Config.Filesystems {
			path, err := fs.Pattern.Substitute(t.Properties)
			if err != nil {
				return nil, err
			}
			fspaths[fs.Id] = FSPath{Id: fs.Id, Path: path}
		}
		t.fspaths = fspaths
	}
	return t.fspaths, nil
}

func (t *Instance) FilesystemPathList() ([]FSPath, error) {
	paths, err := t.FilesystemPaths()
	if err != nil {
		return nil, err
	}
	var list []FSPath
	for _, path := range paths {
		list = append(list, path)
	}
	return FSPathList(list), nil
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

	fspaths, err := t.FilesystemPaths()
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
