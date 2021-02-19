package util

import (
	"errors"

	"strings"

	"melato.org/export/template"
)

/** Implements a simple pattern substitution on strings, using properties and functions
Replaces parenthesized expressions as follows:
(.key) -> Properties[key]
(name) -> Functions[name]()
*/
type Pattern struct {
	Properties map[string]string
	Functions  map[string]func() (string, error)
}

/** Specify a function that is called to get the replacement value. */
func (t *Pattern) SetFunction(key string, f func() (string, error)) {
	if t.Functions == nil {
		t.Functions = make(map[string]func() (string, error))
	}
	t.Functions[key] = f
}

/** Specify the replacement value for (key) */
func (t *Pattern) SetConstant(key string, value string) {
	t.SetFunction(key, func() (string, error) {
		return value, nil
	})
}

func (t *Pattern) Get(key string) (string, error) {
	if strings.HasPrefix(key, ".") {
		pkey := key[1:]
		if t.Properties != nil {
			value, found := t.Properties[pkey]
			if found {
				return value, nil
			}
		}
		return "", errors.New("no such property: " + pkey)
	}
	if t.Functions != nil {
		f, found := t.Functions[key]
		if found {
			value, err := f()
			if err != nil {
				return "", err
			}
			return value, nil
		}
	}
	return "", errors.New("no such function: " + key)
}

func (t *Pattern) Substitute(pattern string) (string, error) {
	tpl, err := template.Paren.NewTemplate(pattern)
	if err != nil {
		return "", err
	}
	return tpl.Applyf(t.Get)
}
