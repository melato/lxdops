package lxdops

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	lxd "github.com/lxc/lxd/client"
	"melato.org/lxdops/util"
)

type LxdClient struct {
	Socket        string
	Project       string `name:"project" usage:"the LXD project to use"`
	rootServer    lxd.InstanceServer
	projectServer lxd.InstanceServer
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

// Server returns the LXD server for the selected project (via the --project flag)
func (t *LxdClient) Server() (lxd.InstanceServer, error) {
	if t.projectServer == nil {
		root, err := t.RootServer()
		if err != nil {
			return nil, err
		}
		if t.Project == "default" || t.Project == "" {
			t.projectServer = root
		} else {
			t.projectServer = root.UseProject(t.Project)
		}
	}
	return t.projectServer, nil
}

// InstanceServer returns an lxd.InstanceServer for the project indicated by name
// name is a composite {project}_{container}, or just {container} for the default project
func (t *LxdClient) InstanceServer(name string) (server lxd.InstanceServer, container string, err error) {
	server, err = t.Server()
	if err != nil {
		return nil, "", err
	}
	project, container := SplitContainerName(name)
	if project != "" {
		server = t.rootServer.UseProject(project)
	}
	return server, container, nil
}

func (t *LxdClient) ContainerServer(name string) (server lxd.InstanceServer, container string, err error) {
	return t.InstanceServer(name)
}

func AnnotateLXDError(name string, err error) error {
	if err == nil {
		return err
	}
	return errors.New(name + ": " + err.Error())
}

func (t *LxdClient) WaitForNetwork(container string) error {
	server, err := t.Server()
	if err != nil {
		return err
	}
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
	server, err := t.Server()
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

func (t *LxdClient) NewExec(name string) *execRunner {
	server, container, err := t.InstanceServer(name)
	return &execRunner{Server: server, Container: container, Error: err}
}

func FileExists(server lxd.InstanceServer, container string, file string) bool {
	reader, _, err := server.GetContainerFile(container, file)
	if err != nil {
		return false
	}
	reader.Close()
	return true
}

func (t *LxdClient) NewPattern(name string) *util.Pattern {
	pattern := &util.Pattern{}
	pattern.SetConstant("container", name)
	pattern.SetConstant("instance", name)
	pattern.SetConstant("", name)
	project := t.Project
	var slashProject string
	if project == "" || project == "default" {
		project = "default"
		slashProject = ""
	} else {
		slashProject = "/" + project
	}
	pattern.SetConstant("project", project)
	pattern.SetConstant("/project", slashProject)
	pattern.SetFunction("lxdparent", func() (string, error) {
		dataset, err := t.GetDefaultDataset()
		if err != nil {
			return "", err
		}
		root := filepath.Dir(dataset)
		if root == "" {
			return "", errors.New("cannot determine zfsroot for dataset: " + dataset)
		}
		return root, nil
	})
	pattern.SetFunction("zfsroot", func() (string, error) {
		dataset, err := t.GetDefaultDataset()
		if err != nil {
			return "", err
		}
		i := strings.Index(dataset, "/")
		if i >= 0 {
			return dataset[0:i], nil
		}
		return dataset, nil
	})
	return pattern
}
