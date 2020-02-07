package lxdops

type OSType interface {
	DefaultVersion() string
	NeedPasswords() bool
	ImageName(version string) string
	InstallPackageCommand(pkg string) string
	AddUserCommand(u *User) []string
	Profiles(version string, apkCache bool) []string
}

func (t *OS) Type() OSType {
	if t.osType == nil {
		if t.IsAlpine() {
			t.osType = &OsTypeAlpine{}
		} else if t.IsUbuntu() {
			t.osType = &OsTypeUbuntu{}
		} else {
			t.osType = nil
		}
	}
	return t.osType
}

func (t *OS) IsAlpine() bool {
	return t.Name == "alpine"
}

func (t *OS) IsUbuntu() bool {
	return t.Name == "ubuntu"
}

///////////////////////
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

func (t *OsTypeUbuntu) DefaultVersion() string { return "18.04" }
func (t *OsTypeUbuntu) ImageName(version string) string {
	if version == "" {
		version = "18.04"
	}
	return "ubuntu:" + version
}
func (t *OsTypeUbuntu) Profiles(version string, apkCache bool) []string {
	return []string{"ubuntu"}
}

///////////////////////
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
func (t *OsTypeAlpine) DefaultVersion() string { return "3.10" }
func (t *OsTypeAlpine) ImageName(version string) string {
	return "images:alpine/" + version
}
func (t *OsTypeAlpine) Profiles(version string, apkCache bool) []string {
	if apkCache {
		return []string{"alpine-" + version}
	} else {
		return nil
	}
}
