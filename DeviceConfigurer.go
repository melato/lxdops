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

func (t *DeviceConfigurer) CreateFilesystem(fs InstanceFS, originDataset string) error {
	if fs.IsDir() {
		return t.CreateDir(fs.Dir(), true)
	}

	script := t.NewScript()
	if originDataset != "" {
		script.Run("sudo", "zfs", "clone", "-p", originDataset, fs.Path)
		return script.Error()
	}

	// create
	args := []string{"zfs", "create", "-p"}
	for key, value := range fs.Filesystem.Zfsproperties {
		args = append(args, "-o", key+"="+value)
	}
	args = append(args, fs.Path)
	script.Run("sudo", args...)
	t.chownDir(script, fs.Dir())
	return script.Error()
}

func (t *DeviceConfigurer) CreateFilesystems(instance, origin *Instance, snapshot string) error {
	paths, err := instance.Filesystems()
	if err != nil {
		return err
	}
	var originPaths map[string]InstanceFS
	if origin != nil {
		originPaths, err = origin.Filesystems()
		if err != nil {
			return err
		}
		for id, path := range paths {
			if !path.IsZfs() {
				return errors.New("cannot use origin with non-zfs filesystem: " + id)
			}
		}
	}
	var pathList []InstanceFS
	for _, path := range paths {
		if origin != nil || !util.DirExists(path.Dir()) {
			pathList = append(pathList, path)
		}
	}
	InstanceFSList(pathList).Sort()

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
	instance := t.Config.NewInstance(name)
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
			originInstance = sourceConfig.NewInstance(parts[0])
			originSnapshot = parts[1]
		}
		if t.Config.DeviceTemplate != "" {
			templateInstance = sourceConfig.NewInstance(t.Config.DeviceTemplate)
		}
	}

	t.CreateFilesystems(instance, originInstance, originSnapshot)

	script := t.NewScript()
	for deviceName, device := range t.Config.Devices {
		dir, err := instance.DeviceDir(deviceName, device)
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
			templateDir, err := templateInstance.DeviceDir(deviceName, device)
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
				fmt.Println("skipping missing template Device: " + deviceName)
			}
		}
		if script.Error() != nil {
			return script.Error()
		}
	}
	return nil
}

func (t *DeviceConfigurer) CreateProfile(name string) error {
	instance := t.Config.NewInstance(name)
	devices := make(map[string]map[string]string)

	for deviceName, device := range t.Config.Devices {
		dir, err := instance.DeviceDir(deviceName, device)
		if err != nil {
			return err
		}
		devices[deviceName] = map[string]string{"type": "disk", "path": device.Path, "source": dir}
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
	oldInstance := t.Config.NewInstance(oldname)
	newInstance := t.Config.NewInstance(newname)
	oldPaths, err := oldInstance.FilesystemList()
	if err != nil {
		return err
	}
	newPaths, err := newInstance.Filesystems()
	if err != nil {
		return err
	}
	s := t.NewScript()
	for _, oldpath := range InstanceFSList(oldPaths).Roots() {
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

func (t *DeviceConfigurer) ListFilesystems(name string) ([]InstanceFS, error) {
	instance := t.Config.NewInstance(name)
	return instance.FilesystemList()
}

func (t *DeviceConfigurer) PrintFilesystems(name string) error {
	instance := t.Config.NewInstance(name)
	filesystems, err := instance.Filesystems()
	if err != nil {
		return err
	}
	writer := &table.FixedWriter{Writer: os.Stdout}
	var fs InstanceFS
	writer.Columns(
		table.NewColumn("FILESYSTEM", func() interface{} { return fs.Id }),
		table.NewColumn("PATH", func() interface{} { return fs.Path }),
		table.NewColumn("PATTERN", func() interface{} { return fs.Filesystem.Pattern }),
	)
	for _, fs = range filesystems {
		writer.WriteRow()
	}
	writer.End()
	return nil
}

func (t *DeviceConfigurer) PrintDevices(name string) error {
	instance := t.Config.NewInstance(name)
	writer := &table.FixedWriter{Writer: os.Stdout}
	var deviceName string
	var d *Device
	writer.Columns(
		table.NewColumn("PATH", func() interface{} { return d.Path }),
		table.NewColumn("SOURCE", func() interface{} {
			dir, err := instance.DeviceDir(name, d)
			if err != nil {
				return err
			}
			return dir
		}),
		table.NewColumn("NAME", func() interface{} { return deviceName }),
		table.NewColumn("FILESYSTEM", func() interface{} { return d.Filesystem }),
		table.NewColumn("DIR", func() interface{} { return d.Dir }),
	)
	for deviceName, d = range t.Config.Devices {
		writer.WriteRow()
	}
	writer.End()
	return nil
}
