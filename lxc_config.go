package lxdops

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxc/lxd/lxc/config"
	"melato.org/lxdops/util"
)

type lxc_config struct {
	currentProject string
}

func (t *lxc_config) configDir() (string, error) {
	configDir := os.Getenv("LXD_CONF")
	if configDir != "" {
		return configDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir = filepath.Join(home, "snap", "lxd", "current", ".config", "lxc")
	if _, err = os.Stat(configDir); err == nil {
		return configDir, nil
	}
	configDir = filepath.Join(home, ".config", "lxc")
	if _, err = os.Stat(configDir); err == nil {
		return configDir, nil
	}
	return "", err
}

func (t *lxc_config) getCurrentProject() (string, error) {
	configDir, err := t.configDir()
	if err != nil {
		return "", err
	}

	var cfg config.Config
	err = util.ReadYaml(filepath.Join(configDir, "config.yml"), &cfg)
	if err != nil {
		return "", err
	}
	local, found := cfg.Remotes["local"]
	if found {
		return local.Project, nil
	}
	return "", nil
}

func (t *lxc_config) CurrentProject() string {
	if t.currentProject == "" {
		project, err := t.getCurrentProject()
		if err != nil || project == "" {
			project = "default"
		}
		t.currentProject = project
		fmt.Fprintf(os.Stderr, "using lxc current project: %s\n", project)
	}
	return t.currentProject
}
