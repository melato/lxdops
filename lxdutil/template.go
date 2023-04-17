package lxdutil

import (
	"fmt"
	"text/template"

	"melato.org/lxdops/util/network"
	"melato.org/lxdops/util/templatecmd"
)

type TemplateOps struct {
	Client *LxdClient `name:"-"`
	templatecmd.TemplateOp
}

type HostFunctions struct {
}

func (h *HostFunctions) Ipv4() (string, error) {
	addresses, err := network.DetectIpv4Addresses(true)
	if err != nil {
		return "", err
	}
	if len(addresses) != 1 {
		return "", fmt.Errorf("got %d public addresses", len(addresses))
	}
	return addresses[0].String(), nil
}

func (h *HostFunctions) Ipv6() (string, error) {
	addresses, err := network.DetectIpv6Addresses(true)
	if err != nil {
		return "", err
	}
	if len(addresses) != 1 {
		return "", fmt.Errorf("got %d public addresses", len(addresses))
	}
	return addresses[0].String(), nil
}

func (t *TemplateOps) Apply() error {
	funcs := make(template.FuncMap)
	funcs["Host"] = func() any { return &HostFunctions{} }
	funcs["Instance"] = func(name string) (any, error) {
		server, err := t.Client.CurrentServer()
		if err != nil {
			return nil, err
		}
		c, _, err := server.GetInstance(name)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	t.TemplateOp.Funcs = funcs
	return t.TemplateOp.Run()
}
