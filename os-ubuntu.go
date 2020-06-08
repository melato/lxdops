package lxdops

type OsTypeUbuntu struct {
	OsTypeDebian
}

func (t *OsTypeUbuntu) ImageName(version string) string {
	return "ubuntu:" + version
}
