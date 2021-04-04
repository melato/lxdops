# lxdops
Go program that launches and configures LXD containers and manages ZFS filesystems attached to these containers as disk devices.
It reads container and filesystem configuration from YAML files.

# Examples
- Create and configure a stopped container that we'll use to clone other containers from:

```
lxdops launch alp.yaml
```

- Create a *dev-test* container by cloning *alp*:
```
lxdops launch dev.yaml
```

- Create @test ZFS snapshots of *dev*'s attached disk devices:

```
lxdops snapshot -s test dev.yaml
```

- Create a *dev-test* the same way that dev was created, but using disk devices cloned from *dev*@test:

```
lxdops launch dev-test.yaml
```

- Rebuild all three containers from the latest LXD image:
```
lxdops rebuild alp.yaml
lxdops rebuild dev-test.yaml
# test dev-test, to make sure all is well, and then rebuild more containers like it:
lxdops rebuild dev.yaml
```

Every container in these examples has its own /home directory as an attached disk device, independent from the container.
If the filesystem/device directory does not exist, it will be created.

The examples are in demo/
```
cd ./demo
```

Before running them, specify a (zfsroot) global property:
```
lxdops property set zfsroot <zfs-filesystem>
```
Replace <zfs-filesystem> with an existing ZFS filesystem (preferably empty).

It's best to run the examples in a clean LXD project so they don't interfere with any other containers or profiles.
See the *LXD Project Support* section below about how to create a clean project.


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
- specify a /home filesystem to attach to containers.
The location of the filesystem is parameterized by the name of the container (instance), the LXD project (project/) and a global property (zfsroot).

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
- create a zfs filesystem and attach it to the container as /home
- install the specified packages
- stop the container after configuring
- create a container snapshot named "copy"
- we don't create any users in this container.  We'll create users in its clones.

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
- attach a new /home directory
- copy files form alp's /home
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

- clone alp/copy, just like dev
- attach /home, by cloning it from *dev*'s home@test
- add a user, just like dev

# Description

lxdops launches, copies, and deletes *lxdops instances*.

An lxdops **instance** is:
- An LXD container
- A set of ZFS filesystems
- A set of disk devices that are in these filesystems and are attached to the container (via a profile)
- An LXD profile that specifies the attached disk devices

A Yaml instance configuration file specifies how to launch and configure an instance.  It specifies:
- packages to install in the container
- LXD profiles to attach to the container
- ZFS Filesystems to create and zfs properties for these filesystems
- Disk Devices in these filesystems that are attached to the container
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

To create a new clean LXD project, *t1*, you can use an lxdops convenience command:
```
lxdops project create t1
lxc project switch t1
```
This creates a new project with:
- Its own profiles.  lxdops creates a profile for each container
- Shared images.  lxdops does not create, modify, or delete any images
- The default profile copied from the default project

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

Compile:
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
apt install git gcc libc6-dev
```

## Alpine
On Alpine Linux, you may need to do this:
```
apk add git gcc libc-dev linux-headers
```
and compile with static linking, so you can use it on a Debian/Ubuntu LXD host:
```
go install -ldflags -extldflags "-static" lxdops.go
```

# External Programs

lxdops calls these external programs, on the host, with *sudo* when necessary:
- lxc (It mostly uses the LXD API, but uses the "lxc" command for launching and cloning containers)
- zfs
- rsync
- chown
- mkdir
- mv

lxdops calls these external programs, in the container:
- sh
- chpasswd
- chown
- OS-specific commands for adding packages and creating users

