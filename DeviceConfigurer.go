package lxdops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

func (t *DeviceConfigurer) CreateFilesystem(fs *Filesystem, name string) error {
	properties := t.Config.NewProperties(name)
	path, err := fs.Pattern.Substitute(properties)
	if err != nil {
		return err
	}
	if strings.HasPrefix(path, "/") {
		return t.CreateDir(path, true)
	}
	script := t.NewScript()
	if t.Config.DeviceOrigin != "" {
		sourceConfig, err := t.Config.GetSourceConfig()
		if err != nil {
			return err
		}
		parts := strings.Split(t.Config.DeviceOrigin, "@")
		if len(parts) != 2 {
			return errors.New("device origin should be a snapshot: " + t.Config.DeviceOrigin)
		}
		sourceInstance, sourceSnapshot := parts[0], parts[1]
		sourceFS := sourceConfig.Filesystem(fs.Id)
		if sourceFS != nil {
			// clone
			sourceProperties := sourceConfig.NewProperties(sourceInstance)
			sourceDataset, err := sourceFS.Pattern.Substitute(sourceProperties)
			if err != nil {
				return err
			}
			script.Run("sudo", "zfs", "clone", "-p", sourceDataset+"@"+sourceSnapshot, path)
			return script.Error()
		}
	}

	// create
	args := []string{"zfs", "create", "-p"}
	for key, value := range fs.Zfsproperties {
		args = append(args, "-o", key+"="+value)
	}
	args = append(args, path)
	script.Run("sudo", args...)
	t.chownDir(script, filepath.Join("/", path))
	return script.Error()
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

type FSPath string

func (path FSPath) Dir() string {
	if strings.HasPrefix(string(path), "/") {
		return string(path)
	} else {
		return "/" + string(path)
	}
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

func (t *DeviceConfigurer) DeviceDir(properties *util.PatternProperties, filesystems map[string]FSPath, device *Device) (string, error) {
	var fsPath FSPath
	dir, err := device.Dir.Substitute(properties)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(dir, "/") {
		return dir, nil
	}
	if dir == "" {
		dir = device.Name
	} else if device.Dir == "." {
		dir = ""
	}

	fs, err := t.DeviceFilesystem(device)
	if err != nil {
		return "", err
	}
	fsPath = filesystems[fs.Id]
	if dir != "" {
		return filepath.Join(fsPath.Dir(), dir), nil
	} else {
		return fsPath.Dir(), nil
	}
}

func (t *DeviceConfigurer) ConfigureDevices(name string) error {
	filesystems, err := t.Config.FilesystemMap(name)
	if err != nil {
		return err
	}
	for _, fs := range FilesystemList(t.Config.Filesystems).Sorted() {
		fsPath, _ := filesystems[fs.Id]
		if t.Config.DeviceOrigin != "" || !util.DirExists(fsPath.Dir()) {
			err := t.CreateFilesystem(fs, name)
			if err != nil {
				return err
			}
		}
	}
	var templateFilesystems map[string]FSPath
	var templateProperties *util.PatternProperties
	if t.Config.DeviceTemplate != "" {
		sourceConfig, err := t.Config.GetSourceConfig()
		if err != nil {
			return err
		}
		templateProperties = sourceConfig.NewProperties(t.Config.DeviceTemplate)
		templateFilesystems, err = sourceConfig.FilesystemMap(t.Config.DeviceTemplate)
		if err != nil {
			return err
		}
	}
	script := t.NewScript()
	properties := t.Config.NewProperties(name)
	for _, device := range t.Config.Devices {
		dir, err := t.DeviceDir(properties, filesystems, device)
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
			templateDir, err := t.DeviceDir(templateProperties, templateFilesystems, device)
			if err != nil {
				return err
			}
			if util.DirExists(templateDir) {
				script.Run("sudo", "rsync", "-a", templateDir+"/", dir+"/")
			} else {
				fmt.Println("skipping missing Device Template: " + templateDir)
			}
		}
		if script.Error() != nil {
			return script.Error()
		}
	}
	return nil
}

func (t *DeviceConfigurer) CreateProfile(name string) error {
	filesystems, err := t.Config.FilesystemMap(name)
	if err != nil {
		return err
	}
	properties := t.Config.NewProperties(name)
	devices := make(map[string]map[string]string)

	for _, device := range t.Config.Devices {
		dir, err := t.DeviceDir(properties, filesystems, device)
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
	oldproperties := t.Config.NewProperties(oldname)
	newproperties := t.Config.NewProperties(newname)
	s := t.NewScript()
	for _, fs := range RootFilesystems(t.Config.Filesystems) {
		oldpath, err := fs.Pattern.Substitute(oldproperties)
		if err != nil {
			return err
		}
		newpath, err := fs.Pattern.Substitute(newproperties)
		if err != nil {
			return err
		}
		if strings.HasPrefix(oldpath, "/") {
			if util.DirExists(newpath) {
				return errors.New(newpath + ": already exists")
			}
			s.Run("mv", oldpath, newpath)
		} else {
			s.Run("sudo", "zfs", "rename", oldpath, newpath)
		}
	}
	return s.Error()
}

func (t *DeviceConfigurer) ListFilesystems(name string) ([]FSPath, error) {
	filesystems, err := t.Config.FilesystemMap(name)
	if err != nil {
		return nil, err
	}
	var result []FSPath
	for _, fs := range t.Config.Filesystems {
		fsPath, _ := filesystems[fs.Id]
		if util.DirExists(fsPath.Dir()) {
			result = append(result, fsPath)
		}
	}
	return result, nil
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
	pattern := t.Config.NewProperties(name)
	filesystems, err := t.Config.FilesystemMap(name)
	if err != nil {
		return err
	}
	writer := &table.FixedWriter{Writer: os.Stdout}
	var d *Device
	writer.Columns(
		table.NewColumn("PATH", func() interface{} { return d.Path }),
		table.NewColumn("SOURCE", func() interface{} {
			dir, err := t.DeviceDir(pattern, filesystems, d)
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
