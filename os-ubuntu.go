package lxdops

type OsTypeUbuntu struct {
}

func (t *OsTypeUbuntu) NeedPasswords() bool { return false }

func (t *OsTypeUbuntu) InstallPackageCommand(pkg string) string {
	return "DEBIAN_FRONTEND=noninteractive apt-get -y install " + pkg
}

func (t *OsTypeUbuntu) AddUserCommand(u *User) []string {
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

func (t *OsTypeUbuntu) ImageName(version string) string {
	return "ubuntu:" + version
}
func (t *OsTypeUbuntu) Profiles(version string, apkCache bool) []string {
	return []string{"ubuntu"}
}
