package lxdops

import (
	"errors"
	"strconv"

	"melato.org/script/v2"
)

type Ids struct {
	Container string
	uids      map[string]int
	gids      map[string]int
}

func (t *Ids) IsNumber(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '-' {
		s = s[1:]
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (t *Ids) convert(s *script.Script, idString string) int {
	id, err := strconv.Atoi(idString)
	if err != nil {
		s.Errors.Handle(err)
		return -1
	}
	return id
}

func (t *Ids) Uid(s *script.Script, user string) int {
	if s.HasError() {
		return 0
	}
	if t.IsNumber(user) {
		return t.convert(s, user)
	}
	if t.uids == nil {
		t.uids = make(map[string]int)
	}
	id, found := t.uids[user]
	if !found {
		lines := s.Cmd("lxc", "exec", t.Container, "id", "--", "-u", user).ToLines()
		if len(lines) != 1 {
			s.Errors.Clear()
			s.Errors.Handle(errors.New("unknown user: " + user))
			return -1
		}
		id = t.convert(s, lines[0])
		t.uids[user] = id
	}
	return id
}

func (t *Ids) Gid(s *script.Script, group string) int {
	if s.HasError() {
		return 0
	}
	if t.IsNumber(group) {
		return t.convert(s, group)
	}
	if t.uids == nil {
		t.uids = make(map[string]int)
	}
	id, found := t.uids[group]
	if !found {
		lines := s.Cmd("lxc", "exec", t.Container, "id", "--", "-g", group).ToLines()
		if len(lines) != 1 {
			s.Errors.Clear()
			s.Errors.Handle(errors.New("unknown group: " + group))
			return -1
		}
		id = t.convert(s, lines[0])
		t.uids[group] = id
	}
	return id
}
