package lxdops

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"melato.org/lxdops/util"

	lxd "github.com/lxc/lxd/client"
)

func (t *ContainerOps) File(name string, file string) error {
	server, container, err := t.Client.InstanceServer(name)
	if err != nil {
		return err
	}
	reader, response, err := server.GetContainerFile(container, file)
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	util.PrintYaml(response)
	if reader != nil {
		err = reader.Close()
		if err != nil {
			return errors.New(file + ": " + err.Error())
		}
	}
	return nil
}

func (t *ContainerOps) Push(name string, hostFile, containerFile string) error {
	content, err := os.ReadFile(hostFile)
	if err != nil {
		return err
	}
	server, container, err := t.Client.InstanceServer(name)
	if err != nil {
		return err
	}
	var file lxd.ContainerFileArgs
	file.Content = bytes.NewReader(content)
	err = server.CreateContainerFile(container, containerFile, file)
	if err != nil {
		return AnnotateLXDError(containerFile, err)
	}
	return nil
}

func (t *ProjectOps) Use(project ...string) error {
	server, err := t.Client.Server()
	if err != nil {
		return err
	}
	for _, name := range project {
		server, err = t.projectServer(server, name)
		if err != nil {
			return err
		}
		profiles, err := server.GetProfileNames()
		if err != nil {
			return err
		}
		fmt.Printf("project: %s profiles: %d\n", name, len(profiles))

	}
	return nil
}
