#!/bin/sh

# compile a go program statically, so that it can run in any container,
# for example, in alpine containers
mkversion -t version.tpl version.go
CGO_ENABLED=0 go install -ldflags '-extldflags "-static"' lxdops.go version.go
