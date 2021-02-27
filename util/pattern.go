package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"melato.org/export/template"
)

/** Implements a simple pattern substitution on strings, using properties and functions
Replaces parenthesized expressions as follows:
(.key) -> Properties[key]
(name) -> Functions[name]()
*/
type Pattern struct {
	Constants map[string]string
	Functions map[string]func() (string, error)
	didHelp   bool
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
	if t.Constants == nil {
		t.Constants = make(map[string]string)
	}
	t.Constants[key] = value
}

func (t *Pattern) ShowHelp(w io.Writer) {
	fmt.Fprintf(w, "available pattern keys:\n")
	keys := make([]string, 0, len(t.Constants)+len(t.Functions))
	for key, _ := range t.Functions {
		keys = append(keys, key)
	}
	for key, _ := range t.Constants {
		if _, exists := t.Functions[key]; !exists {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, _ := t.Get(key)
		fmt.Fprintf(w, "(%s): %s\n", key, value)
	}
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
	value, found := t.Constants[key]
	if found {
		return value, nil
	}
	if !t.didHelp {
		t.ShowHelp(os.Stderr)
		t.didHelp = true
	}
	return "", errors.New("no such key: " + key)
}

func (t *Pattern) Substitute(pattern string) (string, error) {
	tpl, err := template.Paren.NewTemplate(pattern)
	if err != nil {
		return "", err
	}
	return tpl.Applyf(t.Get)
}
