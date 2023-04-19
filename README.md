lxdops is a go program that launches and configures LXD containers
and manages ZFS filesystems attached to these containers as disk devices.

# Use Case
You want to have a set of containers that have the same guest OS, packages, users, etc.
in order to run websites, for example a container that has a web server, PHP, and several libraries.
You want to have one container per website, but avoid installing and upgrading packages for each container.

You can create a template container with the packages that you want, snapshot it, and then clone it for each website.
You can do this, using lxc copy:
	lxc copy <template-container>/<snapshot> <working-container>
Assuming that you use ZFS or another copy-on-write filesystem, the root filesystem of each working container is a clone of the template container root filesystem, so it uses very little disk space.  Creating the working containers is a relatively fast operation, that does not download and install packages.

But what do you do to upgrade the containers?  If you upgrade each one separately, the working container root filesystems start to diverge from their template.  In addition, you will be downloading and installing the same upgrades multiple times (possibly tens or hundreds of times per LXD host).

lxdops facilitates the following strategy:
- You structure each working container so that application data is not on the root filesystem, but on external disk devices.  Therefore you can replace the root filesystem with a new one, without losing the application data.
- To upgrade, you first create a new template container with the upgrade you want.  Alternately, you can upgrade the existing template container and create a new template snapshot.

Then, for each working container:
- Delete the container
- Clone the container from the template
- re-attach the existing external disk devices
- start the container.  The container should now run with its new OS and the old application data.
This process takes a few seconds per container, during which the container will be offline.

# Operation
## lxdops config file
An lxdops config file is a yaml file that provides instructions about how to build
a template container or a working container.  It can include other config files.
Detailed documentation is in the Go docs:
	cd lxdops
	go doc Config
	
It provides:
- The image or container/snapshot to create the container from.
- A list of cloud-config files to use for configuring the container.
- LXD profiles to attach to the container
- Filesystems and external disk devices to create or use for the container.

# Filesystems/Disk Devices
lxdops can create 0 or more zfs filesystems for each container.
Each filesystem is parameterized by container name, so each container is automatically assigned its own private external filesystems.
If they do not exist, they are created.  If they exist, they are used as is.

The external disk devices that lxdops manages are subdirectories of these filesystems. 
I typically use one filesystem for /var/log, one filesystem for /tmp,
and one filesystem for /var/opt, /etc/opt, /opt, /home, /usr/local/bin.
If they do not already exist, they can be copied from the corresponding devices of a template container.

## cloud-config files
lxdops uses a subset of the cloud-config file format to configure containers internally.
The cloud-config files are applied directly using the LXD API,
without requiring that the container supports cloud-init.
It supports the cloud-init sections: packages, write_files, users, runcmd.

# Goals
- Separate container OS from application data, so that application data
is not on the container root filesystem.
- Create containers by cloning the root filesystem of a template container
(using lxc copy of a snapshot), and creating/copying, or cloninig additional filesystems.

It reads container and filesystem configuration from YAML files.
Configuration is done via yaml files in the cloud-init format (#cloud-config).

# Examples
- Create and configure a stopped container that we'll use to clone other containers from:

```
lxdops launch alp.yaml
```

- Create a *dev* container by cloning *alp*:
```
lxdops launch dev.yaml
```

- Create @test ZFS snapshots of *dev*'s attached disk devices:

```
lxdops snapshot -s test dev.yaml
```

- Make a clone of *dev* for testing

```
lxdops launch dev-test.yaml
```

- Rebuild all three containers from the latest LXD image:
```
lxdops rebuild alp.yaml
lxdops rebuild dev-test.yaml
# test dev-test, to make sure all is well, and then rebuild *dev*:
lxdops rebuild dev.yaml
```
rebuild preserves the container ip address (since 2021-04-05).

Every container in these examples has its own /home filesystem as an attached disk device, independent from the container.

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

If the /home filesystem exists, it will be reused, otherwise it will be created, or cloned and/or copied from another container.


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

dev-test is created the same way as dev, except that its /home filesystem is cloned from the @test snapshot of dev's /home.

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

