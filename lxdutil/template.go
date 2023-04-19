package lxdutil

import (
	"fmt"
	"strconv"
	"text/template"

	"encoding/json"

	"melato.org/lxdops/util/network"
	"melato.org/lxdops/util/templatecmd"
)

type TemplateOps struct {
	Client *LxdClient `name:"-"`
	templatecmd.TemplateOp
}

type Functions struct {
}

func (t *Functions) Json(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type HostFunctions struct {
}

func (h *Functions) Uint16(s string) (uint16, error) {
	d, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint16(d), nil
}

func (h *Functions) Uint8(s string) (uint8, error) {
	d, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint8(d), nil
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

func (h *HostFunctions) PrivateIpv4() (network.Ipv4, error) {
	addresses, err := network.DetectIpv4Addresses(false)
	if err != nil {
		return nil, err
	}
	if len(addresses) != 1 {
		for _, a := range addresses {
			fmt.Printf("%v\n", a)
		}
		return nil, fmt.Errorf("got %d private addresses", len(addresses))
	}
	return addresses[0], nil
}

func (h *HostFunctions) PrivateIpv6() (network.Ipv6, error) {
	var zero network.Ipv6
	addresses, err := network.DetectIpv6Addresses(false)
	if err != nil {
		return zero, err
	}
	if len(addresses) != 1 {
		return zero, fmt.Errorf("got %d private addresses", len(addresses))
	}
	return addresses[0], nil
}

func (t *TemplateOps) Apply() error {
	funcs := make(template.FuncMap)
	funcs["F"] = func() any { return &Functions{} }
	funcs["Host"] = func() any { return &HostFunctions{} }
	funcs["InstanceServer"] = func() (any, error) {
		return t.Client.CurrentServer()
	}
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
	funcs["InstanceState"] = func(name string) (any, error) {
		server, err := t.Client.CurrentServer()
		if err != nil {
			return nil, err
		}
		c, _, err := server.GetInstanceState(name)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	t.TemplateOp.Funcs = funcs
	return t.TemplateOp.Run()
}
