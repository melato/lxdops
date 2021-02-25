package lxdops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
	"melato.org/lxdops/password"
	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type Configurer struct {
	Client        *LxdClient `name:"-"`
	ConfigOptions ConfigOptions
	Trace         bool     `name:"trace,t" usage:"print exec arguments"`
	DryRun        bool     `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Components    []string `name:"components" usage:"which components to configure: packages, scripts, users"`
	All           bool     `name:"all" usage:"If true, configure all parts, except those that are mentioned explicitly, otherwise configure only parts that are mentioned"`
	Packages      bool     `name:"packages" usage:"whether to install packages"`
	Scripts       bool     `name:"scripts" usage:"whether to run scripts"`
	Files         bool     `name:"files" usage:"whether to copy files"`
	Users         bool     `name:"users" usage:"whether to create users and change passwords"`
}

func (t *Configurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *Configurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace, DryRun: t.DryRun}
}

func (t *Configurer) runScriptLines(name string, lines []string) error {
	content := strings.Join(lines, "\n")
	return t.runScript(name, content)
}

func (t *Configurer) runScript(name string, content string) error {
	if t.Trace {
		fmt.Println("sh < ---")
		fmt.Println(content)
		fmt.Println("---")
	}
	return t.Client.NewExec(name).Run(content, "sh")
}

func (t *Configurer) installPackages(config *Config, name string) error {
	if len(config.Packages) == 0 {
		return nil
	}

	var lines []string
	for _, pkg := range config.Packages {
		lines = append(lines, config.OS.Type().InstallPackageCommand(pkg))
	}
	return t.runScriptLines(name, lines)
}

func (t *Configurer) createSnapshot(name, snapshot string) error {
	server, container, err := t.Client.ContainerServer(name)
	if err != nil {
		return err
	}
	op, err := server.CreateContainerSnapshot(container, api.ContainerSnapshotsPost{Name: snapshot})
	if err != nil {
		return AnnotateLXDError(container, err)
	}
	if err := op.Wait(); err != nil {
		return AnnotateLXDError(container, err)
	}
	return nil
}

func (t *Configurer) pushAuthorizedKeys(config *Config, name string) error {
	hostHome, homeExists := os.LookupEnv("HOME")
	if !homeExists {
		return errors.New("host $HOME doesn't exist")
	}
	hostFile := filepath.Join(hostHome, ".ssh", "authorized_keys")
	authorizedKeys, err := os.ReadFile(hostFile)
	if err != nil {
		return err
	}
	server, container, err := t.Client.InstanceServer(name)
	if err != nil {
		return err
	}
	for _, user := range config.Users {
		user = user.EffectiveUser()
		if !user.Ssh {
			continue
		}
		home := user.HomeDir()
		guestFile := filepath.Join(home, ".ssh", "authorized_keys")
		if !FileExists(server, container, guestFile) {
			if t.Trace {
				fmt.Printf("creating %s\n", guestFile)
			}
			file := lxd.ContainerFileArgs{Content: bytes.NewReader(authorizedKeys)}
			err := server.CreateContainerFile(container, guestFile, file)
			if err != nil {
				return AnnotateLXDError(guestFile, err)
			}
			err = t.Client.NewExec(name).Run("", "chown", user.Name+":"+user.Name, guestFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type Sudo struct {
	didSudoers bool
}

func (t *Sudo) Configure(user string) []string {
	var lines []string
	if !t.didSudoers {
		t.didSudoers = true
		lines = append(lines, "mkdir -p /etc/sudoers.d")
	}
	sudoerFile := "/etc/sudoers.d/" + user
	lines = append(lines, "echo '"+user+" ALL=(ALL) NOPASSWD:ALL' > "+sudoerFile)
	lines = append(lines, "chmod 440 "+sudoerFile)
	return lines
}

type Doas struct {
	didSudoers bool
}

func (t *Doas) Configure(user string) []string {
	var lines []string
	lines = append(lines,
		fmt.Sprintf(`if [ -f /etc/doas.conf ]; then  grep "^permit nopass %s$" /etc/doas.conf  || echo "permit nopass %s" >> /etc/doas.conf; fi`,
			user, user))
	return lines
}

func (t *Configurer) createUsers(config *Config, name string) error {
	if config.Users == nil {
		return nil
	}
	var err error
	var sudo Sudo
	var doas Doas
	var lines []string
	hasSsh := false
	for _, user := range config.Users {
		user = user.EffectiveUser()
		if user.Name == "" {
			return errors.New("missing user name")
		}
		var homeDir = user.HomeDir()
		if user.Name != "root" && user.Home != "" {
			parent := filepath.Dir(homeDir)
			if parent != "" {
				lines = append(lines, "mkdir -p "+parent)
			}
		}
		if user.Name != "root" {
			// do not create a "root" user
			// but setup password and authorized_keys later
			args := config.OS.Type().AddUserCommand(user)
			if len(args) == 0 {
				return errors.New("create users is not supported for this os")
			}
			lines = append(lines, util.EscapeShell(args...))
			for _, group := range user.Groups {
				lines = append(lines, "adduser "+user.Name+" "+group)
			}
			if user.Sudo {
				lines = append(lines, sudo.Configure(user.Name)...)
				if config.OS.IsAlpine() {
					lines = append(lines, doas.Configure(user.Name)...)
				}
			}
		}
		if user.Ssh {
			hasSsh = true
			sshDir := homeDir + "/.ssh"
			lines = append(lines, "mkdir -p "+sshDir)
			lines = append(lines, "chown -R "+user.Name+":"+user.Name+" "+sshDir)
			lines = append(lines, "") // this is needed for some reason.
		}
	}
	content := strings.Join(lines, "\n")
	err = t.runScript(name, content)
	if err != nil {
		return err
	}
	if hasSsh {
		err = t.pushAuthorizedKeys(config, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Configurer) changePasswords(config *Config, name string, users []string) error {
	if !config.OS.Type().NeedPasswords() {
		return nil
	}
	if len(users) == 0 {
		return nil
	}

	pass, err := password.Generate(20)
	if err != nil {
		return err
	}
	var lines []string
	for _, user := range users {
		lines = append(lines, user+":"+pass)
	}
	content := strings.Join(lines, "\n")
	return t.Client.NewExec(name).Run(content, "chpasswd")
}

func (t *Configurer) changeUserPasswords(config *Config, name string) error {
	if !config.OS.Type().NeedPasswords() {
		return nil
	}
	var users []string
	for _, u := range config.Users {
		u = u.EffectiveUser()
		users = append(users, u.Name)
	}
	return t.changePasswords(config, name, users)
}

func (t *Configurer) runScripts(name string, scripts []*Script) error {
	server, container, err := t.Client.ContainerServer(name)
	if err != nil {
		return err
	}
	ex := t.Client.NewExec(name)
	for _, script := range scripts {
		ex.Dir(script.Dir)
		ex.Uid(script.Uid)
		ex.Gid(script.Gid)
		if script.File != "" {
			content, err := os.ReadFile(script.File)
			if err != nil {
				return err
			}
			file := lxd.ContainerFileArgs{Content: bytes.NewReader(content),
				Mode: 0555}
			guestFile := filepath.Join("/root", filepath.Base(script.File))
			err = server.CreateContainerFile(container, guestFile, file)
			if err != nil {
				return AnnotateLXDError(guestFile, err)
			}
			err = ex.Run("", guestFile)
			if err != nil {
				return err
			}
			body := strings.TrimSpace(script.Body)
			if body != "" {
				err := ex.Run(body, "sh")
				if err != nil {
					return err
				}
			}
			if script.Reboot {
				op, err := server.UpdateContainerState(container, api.ContainerStatePut{Action: "stop"}, "")
				if err != nil {
					return AnnotateLXDError(container, err)
				}
				if err := op.Wait(); err != nil {
					return AnnotateLXDError(container, err)
				}
				op, err = server.UpdateContainerState(container, api.ContainerStatePut{Action: "start"}, "")
				if err != nil {
					return AnnotateLXDError(container, err)
				}
				if err := op.Wait(); err != nil {
					return AnnotateLXDError(container, err)
				}
			}
		}
	}
	return nil
}

func (t *Configurer) copyFiles(config *Config, name string) error {
	ids := Ids{Exec: t.Client.NewExec(name)}
	// copy any files
	project, container := SplitContainerName(name)
	s := t.NewScript()
	for _, f := range config.Files {
		var path string
		if strings.HasPrefix(f.Path, "/") {
			path = container + f.Path
		} else {
			path = container + "/" + f.Path
		}
		args := []string{"file"}
		args = append(args, ProjectArgs(project)...)
		args = append(args, "push", f.Source, path, "-p")
		if f.Recursive {
			args = append(args, "-r")
		}
		if f.Mode != "" {
			args = append(args, "--mode", f.Mode)
		}

		var uid, gid int
		if f.User != "" {
			if f.Uid != 0 {
				return errors.New("both uid and user are specified")
			}
			uid = ids.Uid(s, f.User)
		}
		if f.Group != "" {
			if f.Gid != 0 {
				return errors.New("both gid and group are specified")
			}
			gid = ids.Gid(s, f.Group)
		}
		// if we do not set --uid, --gid, lxd uses the calling users's uid/gid.
		// If that is the desired behavior, specify uid: -1, gid: -1
		if uid != -1 {
			args = append(args, "--uid", strconv.Itoa(uid))
		}
		if gid != -1 {
			args = append(args, "--gid", strconv.Itoa(gid))
		}
		s.Run("lxc", args...)
		if s.HasError() {
			return errors.New(fmt.Sprintf("failed to copy file %s: %v", f.Source, s.Error()))
		}
	}
	return nil
}

func (t *Configurer) includes(flag bool) bool {
	if t.All {
		return !flag
	} else {
		return flag
	}
}

/** run things inside the container:  install packages, create users, run scripts */
func (t *Configurer) ConfigureContainer(config *Config, name string) error {
	var err error
	if !t.DryRun {
		err := t.Client.WaitForNetwork(name)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Scripts) {
		err = t.runScripts(name, config.PreScripts)
		if err != nil {
			return err
		}
	}

	if t.includes(t.Packages) {
		err = t.installPackages(config, name)
		if err != nil {
			return err
		}
	}

	if t.includes(t.Users) {
		err = t.createUsers(config, name)
		if err != nil {
			return err
		}
		err = t.changeUserPasswords(config, name)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Files) {
		err = t.copyFiles(config, name)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Scripts) {
		err = t.runScripts(name, config.Scripts)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Users) {
		err = t.changePasswords(config, name, config.Passwords)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Configurer) runConfigure(name string, config *Config) error {
	return t.ConfigureContainer(config, name)
}

func (t *Configurer) Run(args []string) error {
	t.Trace = true
	return t.ConfigOptions.Run(args, t.runConfigure)
}
