package lxdutil

import (
	"fmt"
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
		fmt.Printf("%s\n", image.Fingerprint)
		for _, alias := range image.Aliases {
			fmt.Printf("  %s\n", alias.Name)
		}
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
