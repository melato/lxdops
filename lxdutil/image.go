package lxdutil

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"melato.org/script"
)

type ImageOps struct {
	Client *LxdClient `name:"-"`
}

func (t *ImageOps) List() error {
	server, err := t.Client.CurrentServer()
	if err != nil {
		return err
	}
	images, err := server.GetImages()
	if err != nil {
		return err
	}
	for _, image := range images {
		names := make([]string, len(image.Aliases))
		for i, alias := range image.Aliases {
			names[i] = alias.Name
		}
		fmt.Printf("%s %s\n", image.Fingerprint, strings.Join(names, " "))
	}

	return nil
}

func (t *ImageOps) ListFingerprints() error {
	server, err := t.Client.CurrentServer()
	if err != nil {
		return err
	}

	fingerprints, err := server.GetImageFingerprints()
	if err != nil {
		return err
	}
	for _, fp := range fingerprints {
		fmt.Printf("%s\n", fp)
	}
	return nil
}

func (t *ImageOps) imageFilesystem(server lxd.InstanceServer, image *api.Image) (string, error) {
	profiles := image.Profiles
	for _, name := range profiles {
		profile, _, err := server.GetProfile(name)
		if err != nil {
			return "", err
		}
		for _, device := range profile.Devices {
			if device["type"] == "disk" && device["path"] == "/" {
				poolName := device["pool"]
				if poolName != "" {
					pool, _, err := server.GetStoragePool(poolName)
					if err != nil {
						return "", err
					}
					if pool.Driver == "zfs" {
						return pool.Config["source"], nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("cannot find image filesystem")
}

func (t *ImageOps) imageContainers(server lxd.InstanceServer, image *api.Image) error {
	poolFS, err := t.imageFilesystem(server, image)
	if err != nil {
		return err
	}
	imageFS := path.Join(poolFS, "images", image.Fingerprint)

	cmd := exec.Command("zfs", "list", "-o", "name,clones", "-r", "-t", "snapshot", imageFS)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	lines := script.BytesToLines(b)
	for _, line := range lines {
		fmt.Printf("%s\n", line)
	}
	return nil
}

func (t *ImageOps) getImage(server lxd.InstanceServer, name string) (*api.Image, error) {
	image, _, err := server.GetImage(name)
	if err == nil {
		return image, nil
	}
	aliases, err := server.GetImageAliases()
	if err != nil {
		return nil, err
	}
	for _, alias := range aliases {
		if alias.Name == name {
			image, _, err := server.GetImage(alias.Target)
			return image, err
		}
	}
	return nil, fmt.Errorf("image not found: %s", name)
}

func (t *ImageOps) Containers(name string) error {
	server, err := t.Client.CurrentServer()
	if err != nil {
		return err
	}
	image, err := t.getImage(server, name)
	if err != nil {
		return err
	}
	return t.imageContainers(server, image)
}
