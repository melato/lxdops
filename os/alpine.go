package os

import (
	"melato.org/lxdops"
)

type Alpine struct {
}

func (t *Alpine) NeedPasswords() bool { return true }

func (t *Alpine) InstallPackageCommand(pkg string) string {
	return "apk add " + pkg
}

func (t *Alpine) AddUserCommand(u *lxdops.User) []string {
	args := []string{"adduser", "-g", "", "-D"}
	if u.Uid != "" {
		args = append(args, "-u", u.Uid)
	}
	if u.Shell != "" {
		args = append(args, "-s", u.Shell)
	}
	if u.Home != "" {
		args = append(args, "-h", u.Home)
	}
	args = append(args, u.Name)
	return args
}
func (t *Alpine) ImageName(version string) string {
	return "images:alpine/" + version
}
