package lxdops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/lxc/config"
	"melato.org/lxdops/util"
)

type LxdClient struct {
	Socket string
	//Project        string `name:"project" usage:"the LXD project to use.  Overrides Config.Project"`
	rootServer     lxd.InstanceServer
	projectServer  lxd.InstanceServer
	currentProject string
}

func (t *LxdClient) configDir() (string, error) {
	configDir := os.Getenv("LXD_CONF")
	if configDir != "" {
		return configDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir = filepath.Join(home, "snap", "lxd", "current", ".config", "lxc")
	if _, err = os.Stat(configDir); err == nil {
		return configDir, nil
	}
	configDir = filepath.Join(home, ".config", "lxc")
	if _, err = os.Stat(configDir); err == nil {
		return configDir, nil
	}
	return "", err
}

func (t *LxdClient) getCurrentProject() (string, error) {
	configDir, err := t.configDir()
	if err != nil {
		return "", err
	}

	var cfg config.Config
	err = util.ReadYaml(filepath.Join(configDir, "config.yml"), &cfg)
	if err != nil {
		return "", err
	}
	local, found := cfg.Remotes["local"]
	if found {
		return local.Project, nil
	}
	return "", nil
}

func (t *LxdClient) CurrentProject() string {
	if t.currentProject == "" {
		project, err := t.getCurrentProject()
		if err != nil || project == "" {
			project = "default"
		}
		t.currentProject = project
	}
	return t.currentProject
}

func (t *LxdClient) Init() error {
	t.Socket = "/var/snap/lxd/common/lxd/unix.socket"
	return nil
}

func (t *LxdClient) RootServer() (lxd.InstanceServer, error) {
	if t.rootServer == nil {
		// Connect to LXD over the Unix socket
		server, err := lxd.ConnectLXDUnix(t.Socket, nil)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("%s: %s", t.Socket, err.Error()))
		}
		t.rootServer = server
	}
	return t.rootServer, nil
}

func (t *LxdClient) ProjectServer(project string) (lxd.InstanceServer, error) {
	var err error
	if project == "" {
		project = t.CurrentProject()
	}
	server, err := t.RootServer()
	if err != nil {
		return nil, err
	}
	if project == "default" {
		return server, nil
	}
	return server.UseProject(project), nil
}

func AnnotateLXDError(name string, err error) error {
	if err == nil {
		return err
	}
	return errors.New(name + ": " + err.Error())
}

func WaitForNetwork(server lxd.InstanceServer, container string) error {
	for i := 0; i < 30; i++ {
		state, _, err := server.GetContainerState(container)
		if err != nil {
			return AnnotateLXDError(container, err)
		}
		if state == nil {
			continue
		}
		for _, net := range state.Network {
			for _, a := range net.Addresses {
				if a.Family == "inet" && a.Scope == "global" {
					fmt.Println(a.Address)
					return nil
				}
			}
		}
		fmt.Printf("status: %s\n", state.Status)
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not get ip address for: " + container)
}

func (t *LxdClient) GetDefaultDataset() (string, error) {
	server, err := t.RootServer()
	if err != nil {
		return "", err
	}
	pool, _, err := server.GetStoragePool("default")
	if err != nil {
		return "", err
	}
	name := pool.Config["zfs.pool_name"]
	if name == "" {
		return name, errors.New("no zfs.pool_name")
	}
	return name, nil
}

func FileExists(server lxd.InstanceServer, container string, file string) bool {
	reader, _, err := server.GetContainerFile(container, file)
	if err != nil {
		return false
	}
	reader.Close()
	return true
}

func (t *LxdClient) NewProperties(name string, config *Config) *util.PatternProperties {
	properties := &util.PatternProperties{Properties: config.Properties}
	properties.SetConstant("instance", name)
	project := config.Project
	var projectSlash, project_instance string
	if project == "" || project == "default" {
		project = "default"
		projectSlash = ""
		project_instance = name
	} else {
		projectSlash = project + "/"
		project_instance = project + "_" + name
	}
	properties.SetConstant("project", project)
	properties.SetConstant("project/", projectSlash)
	properties.SetConstant("project_instance", project_instance)
	properties.SetFunction("zfsroot", func() (string, error) {
		dataset, err := t.GetDefaultDataset()
		if err != nil {
			return "", err
		}
		i := strings.Index(dataset, "/")
		if i >= 0 {
			return dataset[0:i], nil
		}
		return "", errors.New("the LXD dataset uses root ZFS dataset: " + dataset)
		return dataset, nil
	})
	return properties
}

func (pattern Pattern) Substitute(properties *util.PatternProperties) (string, error) {
	return properties.Substitute(string(pattern))
}
