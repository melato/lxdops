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
type PatternProperties struct {
	Properties map[string]string
	// Functions are used only for keys that are not in Properties
	Functions map[string]func() (string, error)
	didHelp   bool
}

// SetFunction specifies a function that is called to get the replacement value.  It can be overriden by a constant.
func (t *PatternProperties) SetFunction(key string, f func() (string, error)) {
	if t.Functions == nil {
		t.Functions = make(map[string]func() (string, error))
	}
	t.Functions[key] = f
}

// SetConstantFunction specifies a function that evaluates to a constant.  It can be overriden by Constants
func (t *PatternProperties) SetConstant(key string, value string) {
	t.SetFunction(key, func() (string, error) { return value, nil })
}

func (t *PatternProperties) ShowHelp(w io.Writer) {
	fmt.Fprintf(w, "properties:\n")
	keys := make([]string, 0, len(t.Properties)+len(t.Functions))
	for key, _ := range t.Properties {
		keys = append(keys, key)
	}
	for key, _ := range t.Functions {
		if _, exists := t.Properties[key]; !exists {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, _ := t.Get(key)
		fmt.Fprintf(w, "(%s): %s\n", key, value)
	}
}

func (t *PatternProperties) Get(key string) (string, error) {
	value, found := t.Properties[key]
	if found {
		return value, nil
	}
	f, found := t.Functions[key]
	if found {
		value, err := f()
		if err != nil {
			return "", err
		}
		return value, nil
	}
	if !t.didHelp {
		t.ShowHelp(os.Stderr)
		t.didHelp = true
	}
	return "", errors.New("no such key: " + key)
}

func (t *PatternProperties) Substitute(pattern string) (string, error) {
	tpl, err := template.Paren.NewTemplate(pattern)
	if err != nil {
		return "", err
	}
	return tpl.Applyf(t.Get)
}
