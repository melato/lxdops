package lxdops

import (
	"fmt"
)

var OSTypes map[string]OSType

func init() {
	OSTypes = make(map[string]OSType)
}

type OSType interface {
	NeedPasswords() bool
	ImageName(version string) string
	InstallPackageCommand(pkg string) string
	AddUserCommand(u *User) []string
}

func (t *OS) Type() OSType {
	if t.osType == nil {
		osType, exists := OSTypes[t.Name]
		if exists {
			t.osType = osType
		} else {
			fmt.Println("Unknown OS type: " + t.Name)
		}
	}
	return t.osType
}

func (t *OS) IsAlpine() bool {
	return t.Name == "alpine"
}
