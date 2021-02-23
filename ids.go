package lxdops

import (
	"errors"
	"fmt"
	"strconv"

	"melato.org/script/v2"
)

type Ids struct {
	Container string
	uids      *idset
	gids      *idset
}

type idset struct {
	Container string
	Flag      string
	Label     string
	ids       map[string]int
}

func (t *idset) IsNumber(s string) bool {
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

func (t *idset) convert(s *script.Script, idString string) int {
	id, err := strconv.Atoi(idString)
	if err != nil {
		s.Errors.Handle(err)
		return -1
	}
	return id
}

func (t *idset) id(s *script.Script, sid string) int {
	if s.HasError() {
		return 0
	}
	if t.IsNumber(sid) {
		return t.convert(s, sid)
	}
	if t.ids == nil {
		t.ids = make(map[string]int)
	}
	id, found := t.ids[sid]
	if !found {
		lines := s.Cmd("lxc", "exec", t.Container, "id", "--", t.Flag, sid).ToLines()
		if len(lines) != 1 {
			s.Errors.Clear()
			s.Errors.Handle(errors.New(fmt.Sprintf("unknown %s: %s", t.Label, sid)))
			return -1
		}
		id = t.convert(s, lines[0])
		t.ids[sid] = id
	}
	return id
}

func (t *Ids) Uid(s *script.Script, user string) int {
	if t.uids == nil {
		t.uids = &idset{Container: t.Container, Flag: "-u", Label: "user"}
	}
	return t.uids.id(s, user)
}

func (t *Ids) Gid(s *script.Script, group string) int {
	if t.gids == nil {
		t.gids = &idset{Container: t.Container, Flag: "-g", Label: "group"}
	}
	return t.gids.id(s, group)
}
