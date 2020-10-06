package os

type Ubuntu struct {
	Debian
}

func (t *Ubuntu) ImageName(version string) string {
	return "ubuntu:" + version
}
