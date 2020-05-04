package lxdops

type OsTypeAlpine struct {
}

func (t *OsTypeAlpine) NeedPasswords() bool { return true }

func (t *OsTypeAlpine) InstallPackageCommand(pkg string) string {
	return "apk add " + pkg
}

func (t *OsTypeAlpine) AddUserCommand(u *User) []string {
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
func (t *OsTypeAlpine) ImageName(version string) string {
	return "images:alpine/" + version
}
