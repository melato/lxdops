#!/bin/sh

# create containers alp, c1, c1-test, in the current LXD project
# The containers have devices created from zfs filesystems in (zfsroot)
# You must first specify zfsroot as a global property:
#   lxdops property set zfsroot <existing-zfs-filesystem>
lxdops launch alp.yaml 
lxdops launch c1.yaml
lxdops snapshot -s test c1.yaml
lxdops launch c1-test.yaml
