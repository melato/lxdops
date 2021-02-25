package lxdops

import (
	"strings"
)

const DefaultProject = "default"

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
