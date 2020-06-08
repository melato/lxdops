package lxdops

type OSType interface {
	NeedPasswords() bool
	ImageName(version string) string
	InstallPackageCommand(pkg string) string
	AddUserCommand(u *User) []string
}

func (t *OS) Type() OSType {
	if t.osType == nil {
		if t.IsAlpine() {
			t.osType = &OsTypeAlpine{}
		} else if t.IsDebian() {
			t.osType = &OsTypeDebian{}
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

func (t *OS) IsDebian() bool {
	return t.Name == "debian"
}

func (t *OS) IsUbuntu() bool {
	return t.Name == "ubuntu"
}

