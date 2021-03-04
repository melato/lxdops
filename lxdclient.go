package lxdops

import (
	"errors"
	"fmt"
	"time"

	lxd "github.com/lxc/lxd/client"
	"melato.org/lxdops/util"
)

type LxdClient struct {
	Socket string
	//Project        string `name:"project" usage:"the LXD project to use.  Overrides Config.Project"`
	rootServer    lxd.InstanceServer
	projectServer lxd.InstanceServer
	lxc_config
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

func (pattern Pattern) Substitute(properties *util.PatternProperties) (string, error) {
	return properties.Substitute(string(pattern))
}
