package util

import (
	"errors"

	"melato.org/export/template"
)

/** Implements a simple pattern substitution on strings, using properties and functions
Replaces parenthesized expressions as follows:
(.key) -> Properties[key]
(name) -> Functions[name]()
*/
type Pattern struct {
	Functions map[string]func() (string, error)
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
	f, found := t.Functions[key]
	if found {
		value, err := f()
		if err != nil {
			return "", err
		}
		return value, nil
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
