package lxdutil

import (
	"errors"
	"fmt"

	lxd "github.com/lxc/lxd/client"
)

type LxdClient struct {
	Socket string
	//Project        string `name:"project" usage:"the LXD project to use.  Overrides Config.Project"`
	rootServer    lxd.InstanceServer
	projectServer lxd.InstanceServer
	LxcConfig
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
