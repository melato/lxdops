package lxdops

import (
	"strings"
)

const DefaultProject = "default"

func QualifiedContainerName(project string, container string) string {
	if project == DefaultProject {
		return container
	}
	return project + "_" + container
}

type SnapshotName struct {
	Container string
	Snapshot  string
}

func SplitSnapshotName(name string) SnapshotName {
	var r SnapshotName
	i := strings.Index(name, "/")
	if i >= 0 {
		r.Snapshot = name[i+1:]
		r.Container = name[0:i]
	} else {
		r.Container = name
	}
	return r
}
