package lxdops

import (
	"errors"
	"fmt"
	"strings"
	"time"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

const DefaultProject = "default"

func QualifiedContainerName(project string, container string) string {
	if project == DefaultProject {
		return container
	}
	return project + "_" + container
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

func UpdateContainerState(server lxd.InstanceServer, container string, action string) error {
	op, err := server.UpdateContainerState(container, api.ContainerStatePut{Action: action}, "")
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	if err := op.Wait(); err != nil {
		return AnnotateLXDError(container, err)
	}
	return nil
}

func StartContainer(server lxd.InstanceServer, container string) error {
	return UpdateContainerState(server, container, "start")
}

func StopContainer(server lxd.InstanceServer, container string) error {
	return UpdateContainerState(server, container, "stop")
}

func FileExists(server lxd.InstanceServer, container string, file string) bool {
	reader, _, err := server.GetContainerFile(container, file)
	if err != nil {
		return false
	}
	reader.Close()
	return true
}
