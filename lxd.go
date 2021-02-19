package lxdops

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxc/lxd/shared/api"
	"melato.org/script/v2"
)

const DefaultProject = "default"

func ListContainer(name string) (*api.ContainerFull, error) {
	var s script.Script
	project, container := SplitContainerName(name)
	args := []string{"list"}
	args = append(args, ProjectArgs(project)...)
	args = append(args, container, "--format=json")
	output := s.Cmd("lxc", args...).ToBytes()
	if err := s.Error(); err != nil {
		return nil, err
	}
	var containers []*api.ContainerFull
	err := json.Unmarshal(output, &containers)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, errors.New("container not found: " + name)
}

func ListContainersForProject(project string) ([]*api.ContainerFull, error) {
	var scr script.Script
	args := []string{"list"}
	args = append(args, ProjectArgs(project)...)
	args = append(args, "--format=json")
	output := scr.Cmd("lxc", args...).ToBytes()
	if err := scr.Error(); err != nil {
		return nil, err
	}
	var containers []*api.ContainerFull
	err := json.Unmarshal(output, &containers)
	if err != nil {
		return nil, err
	}
	return containers, err
}

func ProjectArgs(project string) []string {
	if project == "" {
		return nil
	}
	return []string{"--project", project}
}

func SplitContainerName(name string) (project string, container string) {
	i := strings.LastIndex(name, "_")
	if i >= 0 {
		return name[0:i], name[i+1:]
	} else {
		return "", name
	}
}

func QualifiedContainerName(project string, container string) string {
	if project == DefaultProject {
		return container
	}
	return project + "_" + container
}

type SnapshotName struct {
	Project   string
	Container string
	Snapshot  string
}

func SplitSnapshotName(name string) SnapshotName {
	var r SnapshotName
	i := strings.Index(name, "/")
	if i >= 0 {
		r.Snapshot = name[i+1:]
		r.Project, r.Container = SplitContainerName(name[0:i])
	} else {
		r.Project, r.Container = SplitContainerName(name)
	}
	return r
}

func WaitForNetwork(name string) error {
	for i := 0; i < 30; i++ {
		c, err := ListContainer(name)
		if err != nil {
			return err
		}
		if c.State != nil {
			for _, net := range c.State.Network {
				for _, a := range net.Addresses {
					if a.Family == "inet" && a.Scope == "global" {
						fmt.Println(a.Address)
						return nil
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not get ip address for: " + name)
}

func GetDefaultDataset() (string, error) {
	script := script.Script{}
	lines := script.Cmd("lxc", "storage", "get", "default", "zfs.pool_name").ToLines()
	if err := script.Error(); err != nil {
		return "", err
	}
	if len(lines) > 0 {
		fs := lines[0]
		return fs, nil
	}
	return "", errors.New("could not get default zfs.pool_name")
}

func ZFSRoot() (string, error) {
	fs, err := GetDefaultDataset()
	if err != nil {
		return "", err
	}
	zfsroot := filepath.Dir(fs)
	return zfsroot, nil
}

func ProfileExists(profile string) bool {
	// Not sure what profile get does, but it returns an error if the profile doesn't exist.
	// "x" is a key.  It doesn't matter what key we use for our purpose.
	script := script.Script{}
	script.Cmd("lxc", "profile", "get", profile, "x").CombineOutput().ToNull()
	return script.Error() == nil
}

func ListProjects() ([]*api.Project, error) {
	var s script.Script
	output := s.Cmd("lxc", "project", "list", "--format=json").ToBytes()
	if err := s.Error(); err != nil {
		return nil, err
	}
	var projects []*api.Project
	err := json.Unmarshal(output, &projects)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func ProjectNames() ([]string, error) {
	projects, err := ListProjects()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(projects))
	for i, project := range projects {
		names[i] = project.Name
	}
	return names, nil
}

func CurrentProject() (string, error) {
	var s script.Script
	output := s.Cmd("lxc", "project", "list", "--format=csv").ToBytes()
	if err := s.Error(); err != nil {
		return "", err
	}
	reader := csv.NewReader(bytes.NewReader(output))
	for {
		row, err := reader.Read()
		if err != nil {
			return "", err
		}
		if len(row) == 0 {
			// should not happen, but just move on.
			continue
		}
		parts := strings.Split(row[0], " ")
		if len(parts) == 2 && parts[1] == "(current)" {
			return parts[0], nil
		}
	}
	return "", errors.New("could not detect current project")
}
