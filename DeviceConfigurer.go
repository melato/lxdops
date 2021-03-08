package lxdops

import (
	"errors"
	"fmt"
	"os"

	"strings"

	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/util"
	"melato.org/script"
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

	var args []string
	if originDataset != "" {
		args = []string{"zfs", "clone", "-p"}
	} else {
		args = []string{"zfs", "create", "-p"}

	}

	// add properties
	for key, value := range fs.Filesystem.Zfsproperties {
		args = append(args, "-o", key+"="+value)
	}

	if originDataset != "" {
		args = append(args, originDataset)
	}
	args = append(args, fs.Path)
	s := t.NewScript()
	s.Run("sudo", args...)
	if originDataset == "" {
		t.chownDir(s, fs.Dir())
	}
	return s.Error()
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
		path.IsNew = true
	}
	return nil
}

func (t *DeviceConfigurer) ConfigureDevices(instance *Instance) error {
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
			originName := parts[0]
			originSnapshot = parts[1]
			if originName == "" {
				originName = instance.SourceName()
			}
			if originName == "" {
				return errors.New("missing device origin name")
			}
			originInstance = sourceConfig.NewInstance(originName)
		}
		if t.Config.DeviceTemplate != "" {
			templateInstance = sourceConfig.NewInstance(t.Config.DeviceTemplate)
		}
	}

	t.CreateFilesystems(instance, originInstance, originSnapshot)
	filesystems, err := instance.Filesystems()
	if err != nil {
		return err
	}

	script := t.NewScript()
	for deviceName, device := range t.Config.Devices {
		dir, err := instance.DeviceDir(deviceName, device)
		if err != nil {
			return err
		}
		fs, found := filesystems[device.Filesystem]
		if !found {
			fmt.Fprintf(os.Stderr, "missing filesystem: %s\n", device.Filesystem)
			continue
		}
		if !fs.IsNew && util.DirExists(dir) {
			continue
		}
		err = t.CreateDir(dir, true)
		if err != nil {
			return err
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

func (t *DeviceConfigurer) CreateProfile(instance *Instance) error {
	devices := make(map[string]map[string]string)

	for deviceName, device := range t.Config.Devices {
		dir, err := instance.DeviceDir(deviceName, device)
		if err != nil {
			return err
		}
		devices[deviceName] = map[string]string{"type": "disk", "path": device.Path, "source": dir}
	}
	profileName, err := instance.ProfileName()
	if err != nil {
		return err
	}
	server, err := t.Client.ProjectServer(t.Config.Project)
	if err != nil {
		return err
	}
	post := api.ProfilesPost{Name: profileName, ProfilePut: api.ProfilePut{
		Devices:     devices,
		Config:      instance.Config.ProfileConfig,
		Description: "lxdops profile"}}
	if t.Trace {
		fmt.Printf("create profile %s:\n", profileName)
		util.PrintYaml(&post)
	}
	if !t.DryRun {
		return server.CreateProfile(post)
	}
	return nil
}

func (t *DeviceConfigurer) RenameFilesystems(oldInstance, newInstance *Instance) error {
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
