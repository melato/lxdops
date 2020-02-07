package lxdops

import (
	"encoding/xml"
	"io/ioutil"
)

type ConfigXml struct {
	XMLName      xml.Name  `xml:"launch"`
	OS           *OS       `xml:"os"`
	Includes     []Include `xml:"include"`
	Require      []Require `xml:"require"`
	HostFS       string    `xml:"host-fs,omitempty" yaml:"host-fs"`
	Devices      []*Device `xml:"device"`
	Repositories []string  `xml:"repository"`
	Packages     []string  `xml:"package" yaml:"packages"`
	Users        []*User   `xml:"user"`
	Scripts      []*Script `xml:"script"`
	Passwords    []string  `xml:"password" yaml:"passwords"`
}

type Include struct {
	File string `xml:"file,attr"`
}

type Require struct {
	File string `xml:"file,attr"`
}

/** Read raw config from xml */
func ReadConfigXml(file string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var x *ConfigXml
	err = xml.Unmarshal(data, &x)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	config.OS = x.OS

	config.Includes = make([]string, 0, len(x.Includes))
	for _, inc := range x.Includes {
		config.Includes = append(config.Includes, inc.File)
	}

	config.RequiredFiles = make([]string, 0, len(x.Require))
	for _, f := range x.Require {
		config.RequiredFiles = append(config.RequiredFiles, f.File)
	}

	config.HostFS = x.HostFS
	config.Devices = x.Devices
	config.Repositories = x.Repositories
	config.Packages = x.Packages
	config.Users = x.Users
	config.Scripts = x.Scripts
	config.Passwords = x.Passwords

	return config, err
}
