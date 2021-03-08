package lxdops

import (
	"sort"
)

type InstanceDevice struct {
	Device *Device
	Name   string
	Source string
}

type InstanceDeviceList []InstanceDevice

func (t InstanceDeviceList) Len() int           { return len(t) }
func (t InstanceDeviceList) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t InstanceDeviceList) Less(i, j int) bool { return t[i].Source < t[j].Source }

func (t InstanceDeviceList) Sort() { sort.Sort(t) }
