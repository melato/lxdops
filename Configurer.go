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
	"melato.org/lxdops/lxdutil"
	"melato.org/lxdops/password"
	"melato.org/lxdops/util"
	"melato.org/script"
)

type Configurer struct {
	Client *lxdutil.LxdClient `name:"-"`
	ConfigOptions
	Trace      bool     `name:"trace,t" usage:"print exec arguments"`
	DryRun     bool     `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Components []string `name:"components" usage:"which components to configure: packages, scripts, users"`
	All        bool     `name:"all" usage:"If true, configure all parts, except those that are mentioned explicitly, otherwise configure only parts that are mentioned"`
	Packages   bool     `name:"packages" usage:"whether to install packages"`
	Scripts    bool     `name:"scripts" usage:"whether to run scripts"`
	Files      bool     `name:"files" usage:"whether to copy files"`
	Users      bool     `name:"users" usage:"whether to create users and change passwords"`
}

func (t *Configurer) Init() error {
	return t.ConfigOptions.Init()
}

func (t *Configurer) Configured() error {
	return t.ConfigOptions.Configured()
}

func (t *Configurer) NewScript() *script.Script {
	return &script.Script{Trace: t.Trace, DryRun: t.DryRun}
}

func (t *Configurer) NewExec(project string, name string) *execRunner {
	server, err := t.Client.ProjectServer(project)
	ex := &execRunner{Server: server, Container: name, Error: err}
	ex.Trace = t.Trace
	ex.DryRun = t.DryRun
	return ex
}

func (t *Configurer) runScriptLines(project string, name string, lines []string) error {
	content := strings.Join(lines, "\n")
	return t.runScript(project, name, content)
}

func (t *Configurer) runScript(project string, name string, content string) error {
	return t.NewExec(project, name).Run(content, "sh")
}

func (t *Configurer) installPackages(config *Config, name string) error {
	if len(config.Packages) == 0 {
		return nil
	}

	var lines []string
	for _, pkg := range config.Packages {
		lines = append(lines, config.OS.Type().InstallPackageCommand(pkg))
	}
	return t.runScriptLines(config.Project, name, lines)
}

func (t *Configurer) pushAuthorizedKeys(config *Config, container string) error {
	hostHome, homeExists := os.LookupEnv("HOME")
	if !homeExists {
		return errors.New("host $HOME doesn't exist")
	}
	hostFile := filepath.Join(hostHome, ".ssh", "authorized_keys")
	authorizedKeys, err := os.ReadFile(hostFile)
	if err != nil {
		return err
	}
	server, err := t.Client.ProjectServer(config.Project)
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
		if !lxdutil.FileExists(server, container, guestFile) {
			if t.Trace {
				fmt.Printf("creating %s\n", guestFile)
			}
			file := lxd.ContainerFileArgs{Content: bytes.NewReader(authorizedKeys), Mode: 0600}
			err := server.CreateContainerFile(container, guestFile, file)
			if err != nil {
				return lxdutil.AnnotateLXDError(guestFile, err)
			}
			err = t.NewExec(config.Project, container).Run("", "chown", user.Name+":"+user.Name, guestFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type Sudo struct {
}

func (t *Sudo) Configure(user string) []string {
	file := fmt.Sprintf("/etc/sudoers.d/%s.conf", user)
	return []string{
		"if [ -d /etc/sudoers.d ]; then",
		fmt.Sprintf("echo '%s ALL=(ALL) NOPASSWD:ALL' > %s", user, file),
		fmt.Sprintf("chmod 440 %s", file),
		"fi",
	}
}

type Doas struct {
}

func (t *Doas) Configure(user string) []string {
	file := fmt.Sprintf("/etc/doas.d/%s.conf", user)
	return []string{
		"if [ -d /etc/doas.d ]; then",
		fmt.Sprintf("echo permit nopass '%s' > %s", user, file),
		fmt.Sprintf("chmod 400 %s", file),
		"fi",
	}
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
	err = t.runScript(config.Project, name, content)
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

func (t *Configurer) chpasswdInput(pass string, users []string) string {
	var lines []string
	for _, user := range users {
		lines = append(lines, user+":"+pass)
	}
	return strings.Join(lines, "\n")
}

func (t *Configurer) changePasswords(config *Config, name string, users []string) error {
	if !config.OS.Type().NeedPasswords() {
		return nil
	}
	if len(users) == 0 {
		return nil
	}

	length := 20
	pass, err := password.Generate(length)
	if err != nil {
		return err
	}
	content := t.chpasswdInput(pass, users)
	ex := t.NewExec(config.Project, name)
	if t.Trace {
		ex.Trace = false // don't display passwords
		ppass := make([]byte, length)
		for i := 0; i < length; i++ {
			ppass[i] = '*'
		}
		fmt.Println("chpasswd < --- (password hidden)")
		fmt.Println(t.chpasswdInput(string(ppass), users))
		fmt.Println("---")
	}
	return ex.Run(content, "chpasswd")
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

func (t *Configurer) runScripts(project, container string, scripts []*Script) error {
	if t.Trace {
		fmt.Printf("running %d scripts\n", len(scripts))
	}
	server, err := t.Client.ProjectServer(project)
	if err != nil {
		return err
	}
	ex := t.NewExec(project, container)
	for _, script := range scripts {
		if t.Trace {
			fmt.Printf("running script: %s\n", script.Name)
		}
		ex.Dir(script.Dir)
		ex.Uid(script.Uid)
		ex.Gid(script.Gid)
		if script.File != "" {
			content, err := os.ReadFile(string(script.File))
			if err != nil {
				return err
			}
			file := lxd.ContainerFileArgs{Content: bytes.NewReader(content),
				Mode: 0555}
			guestFile := filepath.Join("/root", filepath.Base(string(script.File)))
			err = server.CreateContainerFile(container, guestFile, file)
			if err != nil {
				return lxdutil.AnnotateLXDError(guestFile, err)
			}
			err = ex.Run("", guestFile)
			if err != nil {
				return err
			}
		}
		body := strings.TrimSpace(script.Body)
		if body != "" {
			err := ex.Run(body, "sh")
			if err != nil {
				return err
			}
		}
		if script.Reboot {
			err = (lxdutil.InstanceServer{server}).StopContainer(container)
			if err != nil {
				return err
			}
			err = (lxdutil.InstanceServer{server}).StartContainer(container)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Configurer) copyFiles(config *Config, container string) error {
	// copy any files
	ids := Ids{Exec: t.NewExec(config.Project, container)}
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	for _, f := range config.Files {
		if t.Trace {
			fmt.Printf("create file: %s\n", f.Path)
		}
		file, err := os.Open(string(f.Source))
		if err != nil {
			return err
		}
		defer file.Close()
		var args lxd.ContainerFileArgs
		args.Content = file
		if f.Mode != "" {
			mode, err := strconv.ParseInt(f.Mode, 8, 32)
			if err != nil {
				return errors.New("cannot parse mode: " + f.Mode)
			}
			args.Mode = int(mode)
		} else {
			st, err := os.Stat(string(f.Source))
			if err != nil {
				return err
			}
			args.Mode = int(st.Mode().Perm())
		}

		var uid, gid int64
		if f.User != "" {
			if f.Uid != 0 {
				return errors.New("both uid and user are specified")
			}
			uid, err = ids.Uid(f.User)
			if err != nil {
				return err
			}
		}
		if f.Group != "" {
			if f.Gid != 0 {
				return errors.New("both gid and group are specified")
			}
			gid, err = ids.Gid(f.Group)
			if err != nil {
				return err
			}
		}
		// if we do not set --uid, --gid, lxd uses the calling users's uid/gid.
		// If that is the desired behavior, specify uid: -1, gid: -1
		if uid != -1 {
			args.UID = int64(uid)
		}
		if gid != -1 {
			args.GID = int64(gid)
		}
		err = server.CreateContainerFile(container, f.Path, args)
		if err != nil {
			return lxdutil.AnnotateLXDError(f.Path, err)
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
func (t *Configurer) ConfigureContainer(instance *Instance) error {
	config := instance.Config
	container := instance.Container()
	server, err := t.Client.ProjectServer(config.Project)
	if err != nil {
		return err
	}
	if !t.DryRun {
		err := lxdutil.WaitForNetwork(server, container)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Scripts) {
		err = t.runScripts(config.Project, container, config.PreScripts)
		if err != nil {
			return err
		}
	}

	if t.includes(t.Packages) {
		err = t.installPackages(config, container)
		if err != nil {
			return err
		}
	}

	if t.includes(t.Users) {
		err = t.createUsers(config, container)
		if err != nil {
			return err
		}
		err = t.changeUserPasswords(config, container)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Files) {
		err = t.copyFiles(config, container)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Scripts) {
		err = t.runScripts(config.Project, container, config.Scripts)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Users) {
		err = t.changePasswords(config, container, config.Passwords)
		if err != nil {
			return err
		}
	}
	return nil
}
