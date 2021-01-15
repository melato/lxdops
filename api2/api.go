/** Equivalent to"github.com/lxc/lxd/shared/api" for our purposes. */
package api

type ContainerFull struct {
	Name     string   `json:"name"`
	Profiles []string `json:"profiles"`
	State    *State   `json:"state"`
}

type State struct {
	Network map[string]*Network `json:"network"`
}

type Network struct {
	Addresses []*Address `json:"addresses"`
}

type Address struct {
	Address string `json:"address"`
	Family  string `json:"family"`
	Netmask string `json:"netmask"`
	Scope   string `json:"scope"`
}

type Project struct {
	Name string `json:name`
}
