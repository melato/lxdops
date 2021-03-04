package os

import (
	"melato.org/lxdops"
)

type Debian struct {
}

// NeedPasswords returns false for Debian, because we can disable
// the account (therefore disabling password login), and still  use it
// for passwordless ssh login.
func (t *Debian) NeedPasswords() bool { return false }

func (t *Debian) InstallPackageCommand(pkg string) string {
	return "DEBIAN_FRONTEND=noninteractive apt-get -y install " + pkg
}

func (t *Debian) AddUserCommand(u *lxdops.User) []string {
	args := []string{"adduser", u.Name, "--disabled-password", "--gecos", ""}
	if u.Uid != "" {
		args = append(args, "--uid", u.Uid)
	}
	if u.Shell != "" {
		args = append(args, "--shell", u.Shell)
	}
	if u.Home != "" {
		args = append(args, "--home", u.Home)
	}
	return args
}

func (t *Debian) ImageName(version string) string {
	return "images:debian/" + version
}
