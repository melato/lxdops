package lxdops

type OsTypeDebian struct {
}

func (t *OsTypeDebian) NeedPasswords() bool { return false }

func (t *OsTypeDebian) InstallPackageCommand(pkg string) string {
	return "DEBIAN_FRONTEND=noninteractive apt-get -y install " + pkg
}

func (t *OsTypeDebian) AddUserCommand(u *User) []string {
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

func (t *OsTypeDebian) ImageName(version string) string {
	return "images:debian/" + version
}
