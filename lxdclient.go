package lxdops

import (
	"errors"
	"fmt"

	lxd "github.com/lxc/lxd/client"
)

type LxdClient struct {
	Socket string
	server lxd.ContainerServer
}

func (t *LxdClient) Init() error {
	t.Socket = "/var/snap/lxd/common/lxd/unix.socket"
	return nil
}

func (t *LxdClient) Server() (lxd.ContainerServer, error) {
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
