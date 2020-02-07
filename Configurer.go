package lxdops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"melato.org/export/password"
	"melato.org/export/program"
)

type Configurer struct {
	ops        *Ops
	DryRun     bool     `name:"dry-run" usage:"show the commands to run, but do not change anything"`
	Components []string `name:"components" usage:"which components to configure: packages, scripts, users"`
	Packages   bool     `name:"packages" usage:"whether to install packages"`
	Scripts    bool     `name:"scripts" usage:"whether to run scripts"`
	Users      bool     `name:"users" usage:"whether to create users and change passwords"`
	prog       program.Params
}

func NewConfigurer(ops *Ops) *Configurer {
	var t Configurer
	t.ops = ops
	return &t
}

func (t *Configurer) Configured() error {
	t.prog.DryRun = t.DryRun
	t.prog.Trace = t.ops.Trace
	return nil
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
	if s.Op.ops.Trace {
		fmt.Println(content)
	}
	args := []string{"exec"}
	if s.dir != "" {
		args = append(args, "--cwd", s.dir)
	}
	if s.uid != 0 {
		args = append(args, "--user", strconv.Itoa(s.uid))
		args = append(args, "--group", strconv.Itoa(s.gid))
	}
	args = append(args, name)
	args = append(args, execArgs...)
	cmd, err := s.Op.prog.NewProgram("lxc").Cmd(args...)
	if err != nil {
		return err
	}
	if s.Op.DryRun {
		return nil
	}
	if content != "" {
		cmd.Stdin = strings.NewReader(content)
	} else {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
	return t.prog.NewProgram("lxc").Run("snapshot", name, snapshot)
}

func (t *Configurer) pushAuthorizedKeys(config *Config, name string) error {
	hostHome, homeExists := os.LookupEnv("HOME")
	if !homeExists {
		return errors.New("host $HOME doesn't exist")
	}
	fmt.Println("HOME", hostHome)
	hostFile := filepath.Join(hostHome, ".ssh", "authorized_keys")
	for _, user := range config.Users {
		user = user.EffectiveUser()
		if !user.Ssh {
			continue
		}
		home := user.HomeDir()
		guestFile := filepath.Join(home, ".ssh", "authorized_keys")
		err := t.prog.NewProgram("lxc").Run("file", "push", hostFile, name+guestFile)
		if err != nil {
			return err
		}
		err = t.prog.NewProgram("lxc").Run("exec", name, "chown", user.Name+":"+user.Name, guestFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Configurer) createUsers(config *Config, name string) error {
	if config.Users == nil {
		return nil
	}
	var err error
	var didSudoers bool
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
			lines = append(lines, EscapeShell(args...))
			for _, group := range user.Groups {
				lines = append(lines, "adduser "+user.Name+" "+group)
			}
			if user.Sudo {
				if !didSudoers {
					didSudoers = true
					lines = append(lines, "mkdir -p /etc/sudoers.d")
				}
				sudoerFile := "/etc/sudoers.d/" + user.Name
				lines = append(lines, "echo '"+user.Name+" ALL=(ALL) NOPASSWD:ALL' > "+sudoerFile)
				lines = append(lines, "chmod 440 "+sudoerFile)
			}
		}
		if user.Ssh {
			hasSsh = true
			sshDir := homeDir + "/.ssh"
			lines = append(lines, "mkdir -p "+sshDir)
			lines = append(lines, "chown -R "+user.Name+":"+user.Name+" "+sshDir)
			lines = append(lines, "")
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
	cmd, err := t.prog.NewProgram("lxc").Cmd("exec", name, "chpasswd")
	if err != nil {
		return err
	}
	if t.DryRun {
		return nil
	}
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
	for _, script := range config.Scripts {
		if script.First != first {
			continue
		}
		if script.File != "" {
			file, err := t.ops.GetPath(script.File)
			if err != nil {
				return err
			}
			err = t.prog.NewProgram("lxc").Run("file", "push", file, name+"/root/")
			if err != nil {
				fmt.Println(file, err)
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
				err := t.prog.NewProgram("lxc").Run("stop", name)
				if err != nil {
					return err
				}
				err = t.prog.NewProgram("lxc").Run("start", name)
				if err != nil {
					return err
				}
			}
		}
		return nil
	} else {
		return errors.New("failed to copy scripts: " + strings.Join(failedFiles, ","))
	}
}

/** run things inside the container:  install packages, create users, run scripts */
func (t *Configurer) ConfigureContainer(config *Config, name string) error {
	var err error
	if !t.DryRun {
		err := t.ops.waitForNetwork(name)
		if err != nil {
			return err
		}
	}
	if t.Scripts {
		err = t.runScripts(config, name, true)
		if err != nil {
			return err
		}
	}

	if t.Packages {
		err = t.installPackages(config, name)
		if err != nil {
			return err
		}
		err = t.createSnapshot(name, "packages")
		if err != nil {
			return err
		}
	}

	if t.Users {
		err = t.createUsers(config, name)
		if err != nil {
			return err
		}
		err = t.changeUserPasswords(config, name)
		if err != nil {
			return err
		}
	}
	if t.Scripts {
		err = t.runScripts(config, name, false)
		if err != nil {
			return err
		}
	}
	if t.Users {
		err = t.changePasswords(config, name, config.Passwords)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Configurer) Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: {name} {configfile}...")
	}
	name := args[0]
	var err error
	var config *Config
	config, err = ReadConfigs(args[1:]...)
	if err != nil {
		return err
	}

	config.Verify()

	return t.ConfigureContainer(config, name)
}
