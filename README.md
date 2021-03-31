# lxdops
Go program that uses YAML configuration files to launch and configure LXD containers with attached disk devices.

# Examples
To run the examples in a clean LXD project, *t1*:
```
lxdops project create t1
lxc project switch t1
lxdops property set zfsroot z/demo # use any ZFS filesystem that can be created with *zfs create -p*

cd ./demo
lxdops launch alp.yaml
lxdops launch dev.yaml
lxdops snapshot -s test dev.yaml
lxdops launch dev-test.yaml

```

## home.yaml (included by other examples)
```
filesystems:
  main:
    pattern: (zfsroot)/(project/)home/(instance)
devices:
  home:
    path: /home
    filesystem: main
```
- specify a /home filesystem to attach to containers

## user.yaml (included by other examples)
```
users:
    - name: user1
      uid: 2001
      shell: /bin/bash
      sudo: true
      ssh: true
      groups:
        - wheel
        - adm
```
- create a user
- install ~/.ssh/authorized_keys by copying the calling user's ~/.ssh/authorized_keys'

## alp.yaml - launch a container from scratch
```
os:
  name: alpine
  version: 3.13
packages:
- sudo
- bash
include:
- home.yaml
stop: true
snapshot: copy
```
- create an alpine container
- create a zfs filesystem and attach it to the container
- install the specified packages
- stop the container after configuring
- create a container snapshot named "copy"

## dev.yaml - clone a container 
```
os:
  name: alpine
origin: alp/copy
device-template: alp
include:
- home.yaml
- user.yaml

```

- clone the alp/copy container snapshot
- attach a /home directory
- add a user

## dev-test.yaml - clone a container and its attached filesystems
```
os:
  name: alpine
origin: alp/copy
device-origin: dev@test
include:
- home.yaml
- user.yaml
```

- clone the alp/copy container snapshot
- clone the dev home@test filesystem and attach it as /home
- add the user again

# Description

lxdops launches, copies, and deletes *instances*.

An **instance** is:
- An LXD container
- A set of ZFS filesystems
- A set of disk devices that are in these filesystems and are attached to the container (via a profile)

A Yaml instance configuration file specifies how to launch and configure an instance.  It specifies:
- packages to install in the container
- LXD profiles to attach to the container
- ZFS Filesystems to create and zfs properties for those filesystems
- Disk Devices to create and attach to the container
- An LXD profile to create with the instance devices
- scripts to run in the container
- files to push to the container
- users to create in the container, with optional sudo privileges and .ssh/authorized_keys

Several configuration elements can be parameterized with properties such as the instance name, project, and user-defined properties.
This allows test instances to have their devices under a separate test filesystem, etc.

More detailed documentation of configuration elements is in the file Config.go

## LXD Project Support

lxdops has support for LXD projects and can clone instances across projects.
I find it simpler to keep all my instances in a single project.

If an instance does not specify a specific project, lxdops will use the current LXD project, as specified in ~/snap/lxd/current/.config/lxc/config.yml or ~/.config/lxc/config.yml

# More Examples

A more elaborate set of configuration files is provided in a separate repository: https://github.com/melato/lxdops.script

# Build (requires go 1.16)
- go get melato.org/command
- go get melato.org/script
- gopkg.in/yaml.v2
- go get github.com/lxc/lxd
- # ...
- cd main
- date > version
- export GO111MODULE=auto
- go install -ldflags -extldflags "-static" lxdops.go
