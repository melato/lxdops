package lxdops

import (
	"errors"
	"fmt"
	"time"

	lxd "github.com/lxc/lxd/client"
)

type LxdClient struct {
	Socket string
	server lxd.InstanceServer
}

func (t *LxdClient) Init() error {
	t.Socket = "/var/snap/lxd/common/lxd/unix.socket"
	return nil
}

func (t *LxdClient) Server() (lxd.InstanceServer, error) {
	if t.server == nil {
		// Connect to LXD over the Unix socket
		server, err := lxd.ConnectLXDUnix(t.Socket, nil)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("%s: %s", t.Socket, err.Error()))
		}
		t.server = server
	}
	return t.server, nil
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
		server = server.UseProject(project)
		if server == nil {
			return server, container, errors.New("no such project:" + project)
		}
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
