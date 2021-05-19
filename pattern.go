package lxdops

import (
	"melato.org/lxdops/util"
)

// Pattern is a string that is converted via property substitution, before it is used
// Properties are denoted in the pattern via (<key>), where <key> is the property key
// There are built-in properties like instance, project.
// Custom properties are defined in Config.Properties, and override built-in properties
type Pattern string

func (pattern Pattern) Substitute(properties *util.PatternProperties) (string, error) {
	return properties.Substitute(string(pattern))
}
