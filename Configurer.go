package lxdops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"melato.org/lxdops/password"
	"melato.org/lxdops/util"
	"melato.org/script/v2"
)

type Configurer struct {
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

type execRunner struct {
	Op  *Configurer
	dir string
	uid int
	gid int
}

func (t *Configurer) NewExec() *execRunner {
	return &execRunner{Op: t}
}

func (s *execRunner) Dir(dir string) *execRunner {
	s.dir = dir
	return s
}

func (s *execRunner) Uid(uid int) *execRunner {
	s.uid = uid
	return s
}

func (s *execRunner) Gid(gid int) *execRunner {
	s.gid = gid
	return s
}

func (s *execRunner) Run(name, content string, execArgs []string) error {
	project, container := SplitContainerName(name)
	args := append(ProjectArgs(project), "exec")
	if s.dir != "" {
		args = append(args, "--cwd", s.dir)
	}
	if s.uid != 0 {
		args = append(args, "--user", strconv.Itoa(s.uid))
		args = append(args, "--group", strconv.Itoa(s.gid))
	}
	args = append(args, container)
	args = append(args, execArgs...)
	script := &script.Script{Trace: s.Op.Trace, DryRun: s.Op.DryRun}
	cmd := script.Cmd("lxc", args...)
	if content != "" {
		cmd.InputString(content)
	}
	cmd.Run()
	return script.Error()
}

func (t *Configurer) runScript(name string, content string) error {
	s := t.NewExec()
	return s.Run(name, content, []string{"sh"})
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
	project, container := SplitContainerName(name)
	script := t.NewScript()
	script.Run("lxc", append(ProjectArgs(project), "snapshot", container, snapshot)...)
	return script.Error()
}

func (t *Configurer) pushAuthorizedKeys(config *Config, name string) error {
	hostHome, homeExists := os.LookupEnv("HOME")
	if !homeExists {
		return errors.New("host $HOME doesn't exist")
	}
	hostFile := filepath.Join(hostHome, ".ssh", "authorized_keys")
	project, container := SplitContainerName(name)
	for _, user := range config.Users {
		user = user.EffectiveUser()
		if !user.Ssh {
			continue
		}
		home := user.HomeDir()
		guestFile := filepath.Join(home, ".ssh", "authorized_keys")
		path := container + guestFile
		s := &script.Script{Trace: t.Trace, DryRun: t.DryRun}
		projectArgs := ProjectArgs(project)
		s.Cmd("lxc", append(projectArgs, "file", "pull", path, "-")...).CombineOutput().ToNull()
		if s.Error() != nil {
			s.Errors.Clear()
			s.Run("lxc", append(projectArgs, "file", "push", hostFile, path)...)
			s.Run("lxc", append(projectArgs, "exec", container, "chown", user.Name+":"+user.Name, guestFile)...)
			if err := s.Error(); err != nil {
				return err
			}
		} else {
			fmt.Printf("%s already exists\n", path)
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
	script := t.NewScript()
	project, container := SplitContainerName(name)
	cmd := script.Cmd("lxc", append(ProjectArgs(project), "exec", container, "chpasswd")...)
	cmd.Cmd.Stdin = strings.NewReader(content)
	cmd.Run()
	return script.Error()
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

func (t *Configurer) runScripts(config *Config, name string, first bool) error {
	// copy any script files
	var failedFiles []string
	project, container := SplitContainerName(name)
	projectArgs := ProjectArgs(project)
	for _, script := range config.Scripts {
		if script.First != first {
			continue
		}
		if script.File != "" {
			s := t.NewScript()
			args := []string{"file"}
			args = append(args, projectArgs...)
			args = append(args, "push", script.File, container+"/root/")
			s.Run("lxc", args...)
			if s.Error() != nil {
				fmt.Println(script.File, s.Error())
				failedFiles = append(failedFiles, script.File)
			}
		}
	}
	if failedFiles == nil {
		for _, script := range config.Scripts {
			if script.First != first {
				continue
			}
			runner := t.NewExec().Dir(script.Dir).Uid(script.Uid).Gid(script.Gid)
			if script.File != "" {
				baseName := filepath.Base(script.File)
				err := runner.Run(name, "", []string{"/root/" + baseName})
				if err != nil {
					return err
				}
			}
			body := strings.TrimSpace(script.Body)
			if body != "" {
				err := runner.Run(name, body, []string{"sh"})
				if err != nil {
					return err
				}
			}
			if script.Reboot {
				s := t.NewScript()
				s.Run("lxc", append(projectArgs, "stop", container)...)
				s.Run("lxc", append(projectArgs, "start", container)...)
				if s.Error() != nil {
					return s.Error()
				}
			}
		}
		return nil
	} else {
		return errors.New("failed to copy scripts: " + strings.Join(failedFiles, ","))
	}
}

func (t *Configurer) copyFiles(config *Config, name string) error {
	// copy any files
	project, container := SplitContainerName(name)
	var failedFiles []string
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
		if f.Uid != -1 {
			args = append(args, "--uid", strconv.Itoa(f.Uid))
		}
		if f.Gid != -1 {
			args = append(args, "--gid", strconv.Itoa(f.Gid))
		}
		s := t.NewScript()
		s.Run("lxc", args...)
		if s.Error() != nil {
			fmt.Println(f.Source, s.Error())
			failedFiles = append(failedFiles, f.Source)
		}
	}
	if failedFiles != nil {
		return errors.New("failed to copy files: " + strings.Join(failedFiles, ","))
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
		err := WaitForNetwork(name)
		if err != nil {
			return err
		}
	}
	if t.includes(t.Scripts) {
		err = t.runScripts(config, name, true)
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
		err = t.runScripts(config, name, false)
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
