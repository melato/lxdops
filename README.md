# lxdops
Go program to launch and configure LXD containers and attached disk devices

lxdops launches, copies, and deletes *instances*.

An **instance** is:
- An LXD container
- A set of ZFS filesystems
- A set of disk devices that are in these filesystems and are attached to the container

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

The configuration file can include other configuration files, to reuse configuration that is common in several instances.

**LXD Project Support**

lxdops has support for LXD projects, but I find it simpler to keep all my instances in a single project.

If an instance does not specify a specific project, lxdops will use the current LXD project, as specified in ~/snap/lxd/current/.config/lxc/config.yml or ~/.config/lxc/config.yml

**Example Configuration**

A set of configuration files is provided in a separate repository: https://github.com/melato/lxdops.script

**Build**
- cd main
- echo dev > version
- export GO111MODULE=auto
- go install -ldflags -extldflags "-static" lxdops.go
