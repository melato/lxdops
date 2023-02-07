# lxdops demo
This directory contains examples that demonstrate lxdops usage.
You can run these examples in a separate "demo" LXD project:

	lxdops project create demo
	lxc project switch demo
	
# standalone.yaml
Create and configure a standalone container.

	lxdops launch standalone.yaml

- create a container from an image
- attach profiles (just the default in this example)
- install some packages
- create a user with ssh access
- copy a file
- run some scripts

	

# devices.yaml
Create a container with attached zfs devices.

First, specify a zfs filesystem to use.
replace z/demo with the zfs filesystem of your choice:

	zfs create -p z/demo
	lxdops property set demoroot z/demo
	lxdops launch devices.yaml

# devices-copy.yaml
Create a container by cloning container "devices" and its devices.

	lxdops snapshot -s t1 devices.yaml
	lxdops launch devices-copy.yaml
	
List the devices, to see what happened:
	
	zfs list -r -o name,origin -t all z/demo
