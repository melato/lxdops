package lxdops

import (
	"errors"
	"fmt"
	"os"

	"strings"

	"github.com/lxc/lxd/shared/api"
	"melato.org/export/table3"
	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type DeviceConfigurer struct {
	Client  *LxdClient
	Config  *Config
	Trace   bool
	DryRun  bool
	FuncMap map[string]func() (string, error)
}

func NewDeviceConfigurer(client *LxdClient, config *Config) *DeviceConfigurer {
	return &DeviceConfigurer{Client: client, Config: config}
}

func (t *DeviceConfigurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace, DryRun: t.DryRun}
}

func (t *DeviceConfigurer) chownDir(scr *script.Script, dir string) {
	scr.Run("sudo", "chown", "1000000:1000000", dir)
}

func (t *DeviceConfigurer) CreateDir(dir string, chown bool) error {
	if !util.DirExists(dir) {
		script := t.NewScript()
		script.Run("sudo", "mkdir", "-p", dir)
		//err = os.Mkdir(dir, 0755)
		if chown {
			t.chownDir(script, dir)
		}
		return script.Error()
	}
	return nil
}

func (t *DeviceConfigurer) FilesystemPaths(name string) ([]string, error) {
	var result []string
	properties := t.Config.NewProperties(name)
	for _, fs := range t.Config.Filesystems {
		path, err := fs.Pattern.Substitute(properties)
		if err != nil {
			return nil, err
		}
		result = append(result, path)
	}
	return result, nil
}

func (t *DeviceConfigurer) DeviceFilesystem(device *Device) (*Filesystem, error) {
	fs := t.Config.Filesystem(device.Filesystem)
	if fs != nil {
		return fs, nil
	}
	return nil, errors.New("no such filesystem: " + device.Filesystem)
}

func (t *DeviceConfigurer) CreateFilesystem(path FSPath, originDataset string) error {
	if path.IsDir() {
		return t.CreateDir(path.Dir(), true)
	}

	script := t.NewScript()
	if originDataset != "" {
		script.Run("sudo", "zfs", "clone", "-p", originDataset, path.Path)
		return script.Error()
	}

	// create
	args := []string{"zfs", "create", "-p"}
	fs := t.Config.Filesystem(path.Id)
	for key, value := range fs.Zfsproperties {
		args = append(args, "-o", key+"="+value)
	}
	args = append(args, path.Path)
	script.Run("sudo", args...)
	t.chownDir(script, path.Dir())
	return script.Error()
}

func (t *DeviceConfigurer) CreateFilesystems(instance, origin *Instance, snapshot string) error {
	paths, err := instance.FilesystemPaths()
	if err != nil {
		return err
	}
	var originPaths map[string]FSPath
	if origin != nil {
		originPaths, err = origin.FilesystemPaths()
		if err != nil {
			return err
		}
		for id, path := range paths {
			if !path.IsZfs() {
				return errors.New("cannot use origin with non-zfs filesystem: " + id)
			}
		}
	}
	var pathList []FSPath
	for _, path := range paths {
		if origin != nil || !util.DirExists(path.Dir()) {
			pathList = append(pathList, path)
		}
	}
	FSPathList(pathList).Sort()

	for _, path := range pathList {
		var originDataset string
		if path.IsZfs() {
			originPath, exists := originPaths[path.Id]
			if exists {
				originDataset = originPath.Path + "@" + snapshot
			}
		}
		err := t.CreateFilesystem(path, originDataset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *DeviceConfigurer) ConfigureDevices(name string) error {
	instance, err := t.Config.NewInstance(name)
	if err != nil {
		return err
	}
	var originInstance *Instance
	var templateInstance *Instance
	var originSnapshot string
	if t.Config.DeviceOrigin != "" || t.Config.DeviceTemplate != "" {
		sourceConfig, err := t.Config.GetSourceConfig()
		if err != nil {
			return err
		}
		if t.Config.DeviceOrigin != "" {
			parts := strings.Split(t.Config.DeviceOrigin, "@")
			if len(parts) != 2 {
				return errors.New("device origin should be a snapshot: " + t.Config.DeviceOrigin)
			}
			originInstance, err = sourceConfig.NewInstance(parts[0])
			if err != nil {
				return err
			}
			originSnapshot = parts[1]
		}
		if t.Config.DeviceTemplate != "" {
			templateInstance, err = sourceConfig.NewInstance(t.Config.DeviceTemplate)
			if err != nil {
				return err
			}
		}
	}

	t.CreateFilesystems(instance, originInstance, originSnapshot)

	script := t.NewScript()
	for _, device := range t.Config.Devices {
		dir, err := instance.DeviceDir(device.Name, device)
		if err != nil {
			return err
		}
		if t.Config.DeviceOrigin == "" {
			err = t.CreateDir(dir, true)
			if err != nil {
				return err
			}
		}
		if t.Config.DeviceTemplate != "" {
			templateDir, err := templateInstance.DeviceDir(device.Name, device)
			if err != nil {
				return err
			}
			if templateDir != "" {
				if util.DirExists(templateDir) {
					script.Run("sudo", "rsync", "-a", templateDir+"/", dir+"/")
				} else {
					fmt.Println("skipping missing Device Template: " + templateDir)
				}
			} else {
				fmt.Println("skipping missing template Device: " + device.Name)
			}
		}
		if script.Error() != nil {
			return script.Error()
		}
	}
	return nil
}

func (t *DeviceConfigurer) CreateProfile(name string) error {
	instance, err := t.Config.NewInstance(name)
	if err != nil {
		return err
	}
	devices := make(map[string]map[string]string)

	for _, device := range t.Config.Devices {
		dir, err := instance.DeviceDir(device.Name, device)
		if err != nil {
			return err
		}
		devices[device.Name] = map[string]string{"type": "disk", "path": device.Path, "source": dir}
	}
	profileName := t.Config.ProfileName(name)
	server, err := t.Client.ProjectServer(t.Config.Project)
	if err != nil {
		return err
	}
	post := api.ProfilesPost{Name: profileName, ProfilePut: api.ProfilePut{Devices: devices, Description: "lxdops devices"}}
	if t.Trace {
		fmt.Printf("create profile %s:\n", profileName)
		util.PrintYaml(&post)
	}
	if !t.DryRun {
		return server.CreateProfile(post)
	}
	return nil
}

func (t *DeviceConfigurer) RenameFilesystems(oldname, newname string) error {
	oldInstance, err := t.Config.NewInstance(oldname)
	if err != nil {
		return err
	}
	newInstance, err := t.Config.NewInstance(newname)
	if err != nil {
		return err
	}
	oldPaths, err := oldInstance.FilesystemPathList()
	if err != nil {
		return err
	}
	newPaths, err := newInstance.FilesystemPaths()
	if err != nil {
		return err
	}
	s := t.NewScript()
	for _, oldpath := range FSPathList(oldPaths).Roots() {
		newpath := newPaths[oldpath.Id]
		if oldpath.IsDir() {
			newdir := newpath.Dir()
			if util.DirExists(newdir) {
				return errors.New(newdir + ": already exists")
			}
			s.Run("mv", oldpath.Dir(), newdir)
		} else {
			s.Run("sudo", "zfs", "rename", oldpath.Dir(), newpath.Dir())
		}
	}
	return s.Error()
}

func (t *DeviceConfigurer) ListFilesystems(name string) ([]FSPath, error) {
	instance, err := t.Config.NewInstance(name)
	if err != nil {
		return nil, err
	}
	return instance.FilesystemPathList()
}

func (t *DeviceConfigurer) PrintFilesystems(name string) error {
	pattern := t.Config.NewProperties(name)
	writer := &table.FixedWriter{Writer: os.Stdout}
	var fs *Filesystem
	writer.Columns(
		table.NewColumn("FILESYSTEM", func() interface{} { return fs.Id }),
		table.NewColumn("PATH", func() interface{} {
			path, err := fs.Pattern.Substitute(pattern)
			if err != nil {
				return err
			}
			return path
		}),
		table.NewColumn("PATTERN", func() interface{} { return fs.Pattern }),
	)
	for _, fs = range t.Config.Filesystems {
		writer.WriteRow()
	}
	writer.End()
	return nil
}

func (t *DeviceConfigurer) PrintDevices(name string) error {
	instance, err := t.Config.NewInstance(name)
	if err != nil {
		return err
	}
	writer := &table.FixedWriter{Writer: os.Stdout}
	var d *Device
	writer.Columns(
		table.NewColumn("PATH", func() interface{} { return d.Path }),
		table.NewColumn("SOURCE", func() interface{} {
			dir, err := instance.DeviceDir(d.Name, d)
			if err != nil {
				return err
			}
			return dir
		}),
		table.NewColumn("NAME", func() interface{} { return d.Name }),
		table.NewColumn("FILESYSTEM", func() interface{} { return d.Filesystem }),
		table.NewColumn("DIR", func() interface{} { return d.Dir }),
	)
	for _, d = range t.Config.Devices {
		writer.WriteRow()
	}
	writer.End()
	return nil
}
