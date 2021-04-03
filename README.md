# lxdops
Go program that uses YAML configuration files to launch and configure LXD containers with attached disk devices.

# Examples
Run the examples in a clean LXD project.

If you want to create a clean LXD project, *t1*, you can do it as follows:
```
lxdops project create t1
# This creates a new project with its own profiles, and copies the default profile from the default project.

# make *t1* the current project:
lxc project switch t1
```

```
lxdops property set zfsroot z/demo
# Replace "z/demo" with any ZFS filesystem that can be created with "sudo zfs create -p"

cd ./demo
lxdops launch alp.yaml
lxdops launch dev.yaml
lxdops snapshot -s test dev.yaml
lxdops launch dev-test.yaml
# we've created a stopped container *alp*, and then cloned it to two other containers, *dev* and *dev-test*.
# All containers have a separate /home directory.  *dev-test* /home is a clone of *dev* /home

# Rebuild all three containers from the latest LXD image, while keeping /home the same:
lxdops rebuild alp.yaml
lxdops rebuild dev-test.yaml
# test dev-test, to make sure all is well, and then rebuild more containers like it:
lxdops rebuild dev.yaml

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
```
export GO111MODULE=auto
export GOPATH=~/go
export GOBIN=~/bin

go get melato.org/lxdops
# this will clone the lxdops repository from github and all dependencies to $GOPATH/src
```

If you prefer to not use my go get server, something like this also works:
```
mkdir -p $GOPATH/src/melato.org
cd $GOPATH/src/melato.org
git clone https://github.com/melato/command
git clone https://github.com/melato/script
git clone https://github.com/melato/table3
git clone https://github.com/melato/lxdops

go get gopkg.in/yaml.v2
go get github.com/lxc/lxd
```

```

cd $GOPATH/src/melato.org/lxdops/main
date > version
go install lxdops.go

# check that it was built:
~/bin/lxdops version
```
## Debian/Ubuntu
On a minimal Debian or Ubuntu system, you may need to do this, if you get build errors:
```
sudo apt install git gcc libc6-dev
```

## Alpine
On Alpine Linux, you may need to do this:
```
apk add git gcc libc-dev linux-headers
```
and compile with static linking:
```
go install -ldflags -extldflags "-static" lxdops.go
```
