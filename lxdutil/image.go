package lxdutil

import (
	"fmt"
	"strings"
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
