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

func SplitSnapshotName(name string) (container, snapshot string) {
	i := strings.Index(name, "/")
	if i >= 0 {
		return name[0:i], name[i+1:]
	} else {
		return name, ""
	}
}
