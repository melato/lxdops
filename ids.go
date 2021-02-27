package lxdops

import (
	"errors"
	"fmt"
	"strconv"

	"melato.org/script/v2"
)

type Ids struct {
	Exec *execRunner
	uids *idset
	gids *idset
}

type idset struct {
	Exec  *execRunner
	Flag  string
	Label string
	ids   map[string]int64
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

func (t *idset) convert(idString string) (int64, error) {
	return strconv.ParseInt(idString, 10, 64)
}

func (t *idset) id(sid string) (int64, error) {
	if t.IsNumber(sid) {
		return t.convert(sid)
	}
	if t.ids == nil {
		t.ids = make(map[string]int64)
	}
	id, found := t.ids[sid]
	if !found {
		var lines []string
		data, err := t.Exec.Output("", "id", t.Flag, sid)
		if err == nil {
			lines = script.BytesToLines(data)
		}
		if len(lines) != 1 {
			return 0, errors.New(fmt.Sprintf("unknown %s: %s", t.Label, sid))
		}
		id, err = t.convert(lines[0])
		if err != nil {
			return 0, err
		}
		t.ids[sid] = id
	}
	return id, nil
}

func (t *Ids) Uid(user string) (int64, error) {
	if t.uids == nil {
		t.uids = &idset{Exec: t.Exec, Flag: "-u", Label: "user"}
	}
	return t.uids.id(user)
}

func (t *Ids) Gid(group string) (int64, error) {
	if t.gids == nil {
		t.gids = &idset{Exec: t.Exec, Flag: "-g", Label: "group"}
	}
	return t.gids.id(group)
}
