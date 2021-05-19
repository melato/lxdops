package lxdutil

import (
	"errors"
	"fmt"
	"strings"
	"time"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

const DefaultProject = "default"

type InstanceServer struct {
	Server lxd.InstanceServer
}

func SplitSnapshotName(name string) (container, snapshot string) {
	i := strings.Index(name, "/")
	if i >= 0 {
		return name[0:i], name[i+1:]
	} else {
		return name, ""
	}
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

func FileExists(server lxd.InstanceServer, container string, file string) bool {
	reader, _, err := server.GetContainerFile(container, file)
	if err != nil {
		return false
	}
	reader.Close()
	return true
}

func (t InstanceServer) updateContainerState(container string, action string) error {
	op, err := t.Server.UpdateContainerState(container, api.ContainerStatePut{Action: action}, "")
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	if err := op.Wait(); err != nil {
		return AnnotateLXDError(container, err)
	}
	return nil
}

func (t InstanceServer) StartContainer(container string) error {
	return t.updateContainerState(container, "start")
}

func (t InstanceServer) StopContainer(container string) error {
	return t.updateContainerState(container, "stop")
}
func (t InstanceServer) ProfileExists(profile string) bool {
	_, _, err := t.Server.GetProfile(profile)
	if err == nil {
		return true
	}
	return false
}
